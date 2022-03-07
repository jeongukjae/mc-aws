package internal

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
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
	ctx := context.Background()

	log.Println("Pulling image", cfg.Image)
	reader, err := cli.ImagePull(ctx, cfg.Image, types.ImagePullOptions{})
	if err != nil {
		return "", err
	}
	defer reader.Close()
	io.Copy(os.Stdout, reader)

	log.Println("Remove container if exists, container name:", cfg.ContainerName)
	err = removeContainerIfExists(cli, cfg.ContainerName)
	if err != nil {
		return "", err
	}

	log.Println("Create image", cfg.Image, "with name", cfg.ContainerName)
	log.Println("Mount data path to", cfg.HostDataPath)
	containerConfig := &container.Config{
		Image:        cfg.Image,
		Env:          []string{fmt.Sprintf("JAVA_OPTS=%s", cfg.JavaToolsOptions)},
		ExposedPorts: nat.PortSet{nat.Port(cfg.Port): struct{}{}},
		Tty:          true,
		AttachStderr: true,
		AttachStdin:  true,
		AttachStdout: true,
		OpenStdin:    true,
	}
	hostConfig := &container.HostConfig{
		PortBindings: nat.PortMap{nat.Port(cfg.Port): []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: cfg.Port}}},
		Mounts:       []mount.Mount{{Type: mount.TypeBind, Source: cfg.HostDataPath, Target: cfg.DataPath}},
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

func AttachContainer(cli *client.Client, containerId string) (types.HijackedResponse, error) {
	log.Println("Attach to container id", containerId)
	ctx := context.Background()
	return cli.ContainerAttach(ctx, containerId, types.ContainerAttachOptions{
		Stderr: true,
		Stdout: true,
		Stdin:  true,
		Stream: true,
	})
}

func CreateChannelForStdin(con net.Conn) chan<- []byte {
	stdin := make(chan []byte)
	go func(w io.WriteCloser) {
		for {
			data, ok := <-stdin
			if !ok {
				fmt.Println("!ok")
				w.Close()
				return
			}
			w.Write(append(data, '\n'))
		}
	}(con)

	return stdin
}

func WaitUntilContainerNotRunning(cli *client.Client, containerId string) {
	ctx := context.Background()
	statusCh, errCh := cli.ContainerWait(ctx, containerId, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			log.Println(err)
		}
	case <-statusCh:
	}
}

func GetServerStatus(cli *client.Client, containerId string, quit <-chan bool, msg chan<- string) (<-chan bool, <-chan bool) {
	isDone := make(chan bool)
	shouldExit := make(chan bool)
	const threshold = 60

	go (func() {
		nZero := 0

		for {
			time.Sleep(time.Second * 15)

			select {
			case <-quit:
				isDone <- true
				return
			default:
				res := ""

				nCon, err := getEstablishedConnection(cli, containerId)
				if err != nil {
					log.Println("Cannot get established conn,", err)
					res += fmt.Sprintf("Cannot get established conn, %s\n", err)
				} else {
					res += fmt.Sprintf("n con: %d\n", nCon)
				}

				if nCon == 0 {
					nZero += 1
				} else {
					nZero = 0
				}

				log.Println("# member:", nCon, ", nZero:", nZero, "/", threshold)

				msg <- res
				if nZero >= threshold {
					shouldExit <- true
				}
			}
		}
	})()

	return isDone, shouldExit
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

func getEstablishedConnection(cli *client.Client, containerId string) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
	defer cancel()

	resp, err := cli.ContainerExecCreate(ctx, containerId, types.ExecConfig{
		Cmd:          []string{"sh", "-c", "netstat -atn | grep :25565 | grep ESTABLISHED | wc -l"},
		AttachStdout: true,
	})
	if err != nil {
		return 0, err
	}

	waiter, err := cli.ContainerExecAttach(ctx, resp.ID, types.ExecStartCheck{})
	if err != nil {
		return 0, err
	}
	defer waiter.Close()

	var outBuf, errBuf bytes.Buffer
	_, err = stdcopy.StdCopy(&outBuf, &errBuf, waiter.Reader)
	if err != nil {
		return 0, err
	}

	out, err := ioutil.ReadAll(&outBuf)
	if err != nil {
		return 0, err
	}
	res := string(out)
	res = strings.TrimSpace(res)
	return strconv.Atoi(res)
}
