package task

import (
	"context"
	"io"
	"log"
	"os"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/google/uuid"
)

type State int

const (
	Pending State = iota
	Scheduled
	Running
	Completed
	Failed
)

type Task struct {
	ID						uuid.UUID
	ContainerId 	string
	Name					string
	State					State
	Image					string
	Memory				int
	Disk					int
	ExposedPorts	network.PortSet
	PortBindings	map[string]string
	RestartPolicy	string
	StartTIme			time.Time
	FinishTime		time.Time
}

type TaskEvent struct {
	ID 					uuid.UUID
	State				State
	Timestamp		time.Time
	Task				Task
}

type Config struct {
	Name					string
	AttachStdin		bool
	AttachStdout	bool
	AttachStderr	bool
	ExposedPorts	network.PortSet
	Cmd						[]string
	Image					string
	Cpu						float64
	Memory				int64
	Disk					int64
	Env 					[]string
	RestartPolicy	string
}

type Docker struct {
	Client		*client.Client
	Config 		Config
}

type DockerResult struct {
	Error 				error
	Action				string
	ContainerId		string
	Result 				string
}

func NewConfig(t *Task) *Config {
	return &Config{
		Name:						t.Name,
		Image:					t.Image,
		RestartPolicy: 	t.RestartPolicy,
	}
}

func NewDocker(c *Config) *Docker {
	dc, _ := client.New(client.FromEnv)
	return &Docker{
		Client: dc,
		Config:	*c,
	}
}

func (d *Docker) Run() DockerResult {
	ctx := context.Background()
	reader, err := d.Client.ImagePull(ctx, d.Config.Image, client.ImagePullOptions{})
	if err != nil {
		log.Printf("Error pulling image %s: %v\n", d.Config.Image, err)
		return DockerResult{Error: err}
	}
	defer reader.Close()
	_, _ = io.Copy(os.Stdout, reader)

	rp := container.RestartPolicy{
		Name: container.RestartPolicyMode(d.Config.RestartPolicy),
	}

	r := container.Resources{
		Memory: d.Config.Memory,
	}

	cc := container.Config{
		Image: d.Config.Image,
		Tty: false,
		Env: d.Config.Env,
		ExposedPorts: d.Config.ExposedPorts,
		Cmd: d.Config.Cmd,
	}

	hc := container.HostConfig{
		RestartPolicy: rp,
		Resources: r,
		PublishAllPorts: true,
	}

	createOpts := client.ContainerCreateOptions{
		Name: d.Config.Name,
		Config: &cc,
		HostConfig: &hc,
	}

	resp, err := d.Client.ContainerCreate(ctx, createOpts)
	if err != nil {
		log.Printf("Error creating container using image %s: %v\n", d.Config.Image, err)
		return DockerResult{Error: err}
	}

	_, err = d.Client.ContainerStart(ctx, resp.ID, client.ContainerStartOptions{})
	if err != nil {
		log.Printf("Error starting container %s: %v\n", resp.ID, err)
		return DockerResult{Error: err}
	}

	out, err := d.Client.ContainerLogs(ctx, resp.ID, client.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow: 		true,
	})
	if err != nil {
		log.Printf("Error streaming logs for container %s: %v\n", resp.ID, err)
		return DockerResult{
			Error:				err,
			ContainerId: 	resp.ID,
			Action: 			"start",
		}
	}

	go func() {
		defer out.Close()

		_, _ = stdcopy.StdCopy(os.Stdout, os.Stderr, out)
	}()

	return DockerResult{
		Error: 				nil,
		Action: 			"start",
		ContainerId: 	resp.ID,
		Result: 			"success",
	}
}

func (d *Docker) Stop(id string) DockerResult {
	log.Printf("Attempting to stop container %v", id)
	ctx := context.Background()
	_, err := d.Client.ContainerStop(ctx, id, client.ContainerStopOptions{})
	if err != nil {
		log.Printf("Error stopping container %s: %v\n", id, err)
		return DockerResult{Error: err}
	}

	_, err = d.Client.ContainerRemove(ctx, id, client.ContainerRemoveOptions{
		RemoveVolumes: 	true,
		RemoveLinks: 		false,
		Force: 					false,
	})

	return DockerResult{
		Error: 			nil,
		Action: 		"stop",
		Result: 		"success",
	}
}