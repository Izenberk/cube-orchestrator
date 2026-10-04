package main

import (
	"fmt"
	"time"

	"github.com/Izenberk/cube-orchestrator/task"
	"github.com/Izenberk/cube-orchestrator/worker"
	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
)

func main() {
	db := make(map[uuid.UUID]*task.Task)
	w := worker.Worker{
		Queue: 	*queue.New(),
		Db: 		db,
	}

	// 1. Define the task with a valide image and Scheduled state
	t := task.Task{
    ID:       uuid.New(),
    Name:     "Task-1",
    State:    task.Scheduled,
    Image:    "strm/helloworld-http", // Publicly available image
    Memory:   1024,
    Disk:     1,
  }

	// 2. Queue and run task for the first time (Start Container)
	fmt.Println("starting task")
	w.AddTask(t)
	result := w.RunTask()
	if result.Error != nil {
		panic(result.Error)
	}

	t.ContainerId = result.ContainerId
	fmt.Printf("task %s is running in container %s\n", t.ID, t.ContainerId)

	// 3. Keep container alive briefly
	fmt.Println("Sleepy time")
	time.Sleep(time.Second * 30)

	// 4. Update task state to Completed and re-queue (Stop Container)
	fmt.Printf("stopping task %s\n", t.ID)
	t.State = task.Completed
	w.AddTask(t)
	result = w.RunTask()
	if result.Error != nil {
		panic(result.Error)
	}

	fmt.Printf("task %s successfully stopped\n", t.ID)
}