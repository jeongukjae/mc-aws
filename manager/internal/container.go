package internal

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

type MCServerConfig struct {
	Image            string
	JavaToolsOptions string
	Port             string
	ContainerName    string

	DataPath     string
	HostDataPath string
}

func NewDockerClient() (*client.Client, error) {
	return client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
}

func RunMinecraftServerContainer(cli *client.Client, cfg *MCServerConfig) (string, error) {
	hostDataPath, err := makeAbsAndCreateDir(cfg.HostDataPath)
	if err != nil {
		return "", err
	}
	ctx := context.Background()

	log.Println("Pulling image", cfg.Image)
	reader, err := cli.ImagePull(ctx, cfg.Image, types.ImagePullOptions{})
	if err != nil {
		return "", err
	}
	defer reader.Close()

	log.Println("Remove container if exists, container name:", cfg.ContainerName)
	err = removeContainerIfExists(cli, cfg.ContainerName)
	if err != nil {
		return "", err
	}

	log.Println("Create image", cfg.Image, "with name", cfg.ContainerName)
	log.Println("Mount data path to", hostDataPath)
	containerConfig := &container.Config{
		Image:        cfg.Image,
		Env:          []string{fmt.Sprintf("JAVA_TOOL_OPTIONS=%s", cfg.JavaToolsOptions)},
		ExposedPorts: nat.PortSet{nat.Port(cfg.Port): struct{}{}},
		Tty:          true,
		AttachStderr: true,
		AttachStdin:  true,
		AttachStdout: true,
		OpenStdin:    true,
	}
	hostConfig := &container.HostConfig{
		PortBindings: nat.PortMap{nat.Port(cfg.Port): []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: cfg.Port}}},
		Mounts:       []mount.Mount{{Type: mount.TypeBind, Source: hostDataPath, Target: cfg.DataPath}},
	}
	resp, err := cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, cfg.ContainerName)
	if err != nil {
		return "", err
	}

	log.Println("Start container", cfg.ContainerName)
	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return "", err
	}

	return resp.ID, nil
}

func AttachContainer(cli *client.Client, containerId string) {
	log.Println("Attach to container id", containerId)
	ctx := context.Background()
	waiter, err := cli.ContainerAttach(ctx, containerId, types.ContainerAttachOptions{
		Stderr: true,
		Stdout: true,
		Stdin:  true,
		Stream: true,
	})

	go io.Copy(os.Stdout, waiter.Reader)
	go io.Copy(os.Stderr, waiter.Reader)

	if err != nil {
		log.Fatal(err)
	}

	inout := make(chan []byte)
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			inout <- []byte(scanner.Text())
		}
	}()

	go func(w io.WriteCloser) {
		for {
			data, ok := <-inout
			//log.Println("Received to send to docker", string(data))
			if !ok {
				fmt.Println("!ok")
				w.Close()
				return
			}

			w.Write(append(data, '\n'))
		}
	}(waiter.Conn)

	statusCh, errCh := cli.ContainerWait(ctx, containerId, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			log.Fatal(err)
		}
	case <-statusCh:
	}
}

func makeAbsAndCreateDir(path string) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	result := filepath.Join(cwd, path)
	err = os.MkdirAll(result, os.ModePerm)
	if err != nil {
		return "", err
	}

	return result, nil
}

func removeContainerIfExists(cli *client.Client, containerName string) error {
	ctx := context.Background()
	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return err
	}

	for _, container := range containers {
		for _, name := range container.Names {
			if name == containerName || name == "/"+containerName {
				log.Println("Found contianer", container.ID)

				err = cli.ContainerRemove(ctx, container.ID, types.ContainerRemoveOptions{
					RemoveVolumes: true,
				})
				return err
			}
		}
	}

	return nil
}
