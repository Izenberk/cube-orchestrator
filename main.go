package main

import (
	"fmt"
	"time"
	"os"

	"github.com/Izenberk/cube-orchestrator/manager"
	"github.com/Izenberk/cube-orchestrator/node"
	"github.com/Izenberk/cube-orchestrator/task"
	"github.com/Izenberk/cube-orchestrator/worker"
	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
	"github.com/moby/moby/client"
)

func main() {
	t := task.Task{
		ID:				uuid.New(),
		Name:			"Task-1",
		State:		task.Pending,
		Image:		"Image-1",
		Memory:		1024,
		Disk:			1,
	}

	te := task.TaskEvent{
		ID:					uuid.New(),
		State:			task.Pending,
		Timestamp: 	time.Now(),
		Task: 			t,
	}

	fmt.Printf("task: %v\n", t)
	fmt.Printf("task event: %v\n", te)

	w := worker.Worker{
		Name: 	"worker-1",
		Queue: 	*queue.New(),
		Db:			make(map[uuid.UUID]*task.Task),
	}
	fmt.Printf("worker: %v\n", w)
	w.CollectStates()
	w.RunTask()
	w.StartTask()
	w.StopTask()

	m := manager.Manager{
		Pending: 	*queue.New(),
		TaskDb: 	make(map[string][]*task.Task),
		EventDb: 	make(map[string][]*task.TaskEvent),
	}

	fmt.Printf("manager: %v\n", m)
	m.SelectWorker()
	m.UpdateTasks()
	m.SendWork()

	n := node.Node{
		Name: 	"Node-1",
		Ip:			"192.168.1.1",
		Cores: 	4,
		Memory: 1024,
		Disk: 	25,
		Role: 	"worker",
	}

	fmt.Printf("node: %v\n", n)

	fmt.Printf("create a test container\n")
	dockerTask, createResult, err := createContainer()
	if err != nil {
		fmt.Printf("Error during container creation: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Sleeping for 5 seconds to observe running container...")
	time.Sleep(time.Second * 5)

	fmt.Printf("Stopping container %s...\n", createResult.ContainerId)
	stopResult, err := stopContainer(dockerTask, createResult.ContainerId)
	if err != nil {
		fmt.Printf("Error during container stop: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Container %s cleanly stopped!\n", stopResult.ContainerId)
}

func createContainer() (*task.Docker, *task.DockerResult, error) {
	// Unique name prevents Docker 409 name conflicts when re-running
	uniqueName := fmt.Sprintf("test-container-%s", uuid.New().String()[:8])

	c := task.Config{
		Name:  uniqueName,
		Image: "postgres:13",
		Env: []string{
			"POSTGRES_USER=cube",
			"POSTGRES_PASSWORD=secret",
		},
	}

	dc, err := client.New(client.FromEnv)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	d := task.Docker{
		Client: dc,
		Config: c,
	}

	result := d.Run()
	if result.Error != nil {
		return nil, nil, fmt.Errorf("docker run failed: %w", result.Error)
	}

	fmt.Printf("Container %s (%s) is running\n", result.ContainerId, uniqueName)
	return &d, &result, nil
}

func stopContainer(d *task.Docker, id string) (*task.DockerResult, error) {
	result := d.Stop(id)
	if result.Error != nil {
		return nil, fmt.Errorf("docker stop failed: %w", result.Error)
	}

	return &result, nil
}