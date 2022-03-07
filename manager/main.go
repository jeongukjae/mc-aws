package main

import (
	"bufio"
	"flag"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/seqsense/s3sync"

	"mc-aws-manager/internal"
)

func main() {
	var (
		image            = flag.String("image", "ghcr.io/jeongukjae/mc-aws:1.18.2", "")
		containerName    = flag.String("container_name", "mc-server-aws", "container name")
		javaToolsOptions = flag.String("java_options", "-Xms1280M -Xmx1280M", "")
		port             = flag.String("port", "25565", "port to use")
		dataPath         = flag.String("data_path", "/mc-server-data", "data path in container")
		hostDataPath     = flag.String("data", "/mc-server-data", "data path in host")
		webhookUrl       = flag.String("webhook", "", "webhook url")
		region           = flag.String("region", "ap-northeast-2", "region of aws resource")
		s3DataPath       = flag.String("s3_path", "", "s3 data path")
		withS3Sync       = flag.Bool("sync", true, "sync s3")
	)
	flag.Parse()

	var sess *session.Session
	var syncManager *s3sync.Manager

	// s3 sync
	if *withS3Sync {
		log.Println("Sync from s3", *withS3Sync)
		sess, _ = session.NewSession(&aws.Config{Region: aws.String(*region)})
		syncManager = s3sync.New(sess)
		syncManager.Sync(*s3DataPath, *hostDataPath)
	}

	// create and run
	cli, err := internal.NewDockerClient()
	if err != nil {
		log.Fatal("Cannot connect to docker", err)
	}
	containerId, err := internal.RunMinecraftServerContainer(cli, &internal.MCServerConfig{
		Port:             *port,
		JavaToolsOptions: *javaToolsOptions,
		Image:            *image,
		ContainerName:    *containerName,
		DataPath:         *dataPath,
		HostDataPath:     *hostDataPath,
	})
	if err != nil {
		log.Fatal("Cannot create container", err)
	}

	// attach and subscribe
	quit := make(chan bool)
	botMsg := make(chan string)

	waiter, err := internal.AttachContainer(cli, containerId)
	isDoneMonitor, shouldExit := internal.GetServerStatus(cli, containerId, quit, botMsg)
	isDoneLogger := internal.SubscribeForWebhook(waiter.Reader, *webhookUrl, quit, botMsg)
	if err != nil {
		log.Fatal(err)
	}
	mcCmdChan := internal.CreateChannelForStdin(waiter.Conn)
	isDoneHttp := internal.RunHttpServer(mcCmdChan, quit)
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			mcCmdChan <- []byte(scanner.Text())
		}
	}()
	go func() {
		<-shouldExit
		mcCmdChan <- []byte("exit")
	}()

	// shutdown
	internal.WaitUntilContainerNotRunning(cli, containerId)
	log.Println("Docker exited")

	quit <- true
	quit <- true
	quit <- true

	<-isDoneLogger
	log.Println("Logger exited")
	<-isDoneMonitor
	log.Println("Monitor exited")
	<-isDoneHttp
	log.Println("Http server exited")

	// s3 sync again
	if *withS3Sync {
		log.Println("Sync to s3")
		syncManager.Sync(*hostDataPath, *s3DataPath)
	}
}
