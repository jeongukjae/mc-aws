package main

import (
	"flag"
	"log"

	"mc-aws-manager/internal"
)

func main() {
	var (
		image            = flag.String("image", "ghcr.io/jeongukjae/mc-aws:1.18.2", "")
		containerName    = flag.String("container_name", "mc-server-aws", "container name")
		javaToolsOptions = flag.String("java_options", "-Xms1536M -Xmx1536M", "")
		port             = flag.String("port", "25565", "port to use")
		dataPath         = flag.String("data_path", "/mc-server-data", "data path in container")
		hostDataPath     = flag.String("data", "/mc-server-data", "data path in host")
	)
	flag.Parse()

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

	internal.AttachContainer(cli, containerId)
}
