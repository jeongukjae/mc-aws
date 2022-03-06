package main

import (
	"bufio"
	"flag"
	"log"
	"os"

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
	)
	flag.Parse()

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
	waiter, err := internal.AttachContainer(cli, containerId)
	isDoneLogger := internal.SubscribeForWebhook(waiter.Reader, *webhookUrl, quit)
	if err != nil {
		log.Fatal(err)
	}
	inout := internal.CreateChannelForStdin(waiter.Conn)
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			inout <- []byte(scanner.Text())
		}
	}()

	// shutdown
	internal.WaitUntilContainerNotRunning(cli, containerId)
	quit <- true
	<-isDoneLogger
}
