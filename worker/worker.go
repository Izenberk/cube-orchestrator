package worker

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/Izenberk/cube-orchestrator/stats"
	"github.com/Izenberk/cube-orchestrator/store"
	"github.com/Izenberk/cube-orchestrator/task"
	"github.com/golang-collections/collections/queue"
)

type Worker struct {
	Name				string
	Queue 			queue.Queue
	Db 					store.Store
	TaskCount		int
	Stats 			*stats.Stats
}

func (w *Worker) CollectStats() {
	for {
			log.Println("Collection stats")
			w.Stats = stats.GetStats()
			w.Stats.TaskCount = w.TaskCount
			time.Sleep(15 * time.Second)
	}
}

func New(name string, taskDbType string) *Worker {
	w := Worker{
		Name: 	name,
		Queue: 	*queue.New(),
	}

	var s store.Store
	switch taskDbType {
	case "memory":
		s = store.NewInMemoryTaskStore()
	}
	w.Db = s
	return &w
}

func (w *Worker) runTask() task.DockerResult {
	t := w.Queue.Dequeue()
	if t == nil {
		log.Println("No tasks in queue")
		return task.DockerResult{Error: nil}
	}

	taskQueued := t.(task.Task)

	err := w.Db.Put(taskQueued.ID.String(), &taskQueued)
	if err != nil {
		msg := fmt.Errorf("error storing task %s: %v", taskQueued.ID.String(), err)
		log.Println(msg)
		return task.DockerResult{Error: msg}
	}

	result, err := w.Db.Get(taskQueued.ID.String())
	if err != nil {
		msg := fmt.Errorf("error getting task %s from database: %v", taskQueued.ID.String(), err)
		log.Println(msg)
		return task.DockerResult{Error: msg}
	}

	taskPersisted := *result.(*task.Task)

	if taskPersisted.State == task.Completed {
		return w.StopTask(taskPersisted)
	}

	var dockerResult task.DockerResult
	if task.ValidStateTransition(
		taskPersisted.State, taskQueued.State) {
			switch taskQueued.State {
			case task.Scheduled:
				dockerResult = w.StartTask(taskQueued)
			case task.Completed:
				dockerResult = w.StopTask(taskQueued)
			default:
				dockerResult.Error = errors.New("We should not get here")
			}
		} else {
			err := fmt.Errorf("Invalid transition from %v to %v", taskPersisted.State, taskQueued.State)
			dockerResult.Error = err
		}

		return dockerResult
}

func (w *Worker) RunTasks() {
	for {
		if w.Queue.Len() != 0 {
			result := w.runTask()
			if result.Error != nil {
				log.Printf("Error running task: %v\n", result.Error)
			}
		} else {
			log.Printf("No tasks to process currently. \n")
		}
		log.Println("Sleeping for 10 seconds.")
		time.Sleep(10 * time.Second)
	}
}

func (w *Worker) StartTask(t task.Task) task.DockerResult {
	config := task.NewConfig(&t)
	d := task.NewDocker(config)
	result := d.Run()
	if result.Error != nil {
		log.Printf("Err running task %v: %v\n", t.ID, result.Error)
		t.State = task.Failed
		w.Db.Put(t.ID.String(), &t)
		return result
	}

	t.ContainerID = result.ContainerID
	t.State = task.Running
	w.Db.Put(t.ID.String(), &t)

	return result
}

func (w *Worker) StopTask(t task.Task) task.DockerResult {
	config := task.NewConfig(&t)
	d := task.NewDocker(config)

	stopResult := d.Stop(t.ContainerID)
	if stopResult.Error != nil {
		log.Printf("Error stopping container %v: %v\n", t.ContainerID, stopResult.Error)
	}
	removeResult := d.Remove(t.ContainerID)
	if removeResult.Error != nil {
		log.Printf("%v\n", removeResult.Error)
	}

	t.FinishTime = time.Now().UTC()
	t.State = task.Completed
	w.Db.Put(t.ID.String(), &t)
	log.Printf("Stopped and removed container %v for task %v\n", t.ContainerID, t.ID)
	return removeResult
}

func (w *Worker) AddTask(t task.Task) {
	w.Queue.Enqueue(t)
}

func (w *Worker) GetTasks() []*task.Task {
	taskList, err := w.Db.List()
	if err != nil {
		log.Printf("error getting list of tasks: %v\n", err)
		return nil
	}

	return taskList.([]*task.Task)
}

func (w *Worker) InspectTask(t task.Task) task.DockerInspectResponse {
	config := task.NewConfig(&t)
	d := task.NewDocker(config)
	return d.Inspect(t.ContainerID)
}

func (w *Worker) UpdateTasks() {
	for {
		log.Println("Checking status of tasks")
		w.updateTasks()
		log.Println("Task updates completed")
		log.Println("Sleeping for 15 seconds")
		time.Sleep(15 * time.Second)
	}
}

func (w *Worker) updateTasks() {
	tasks, err := w.Db.List()

	if err != nil {
		log.Printf("error getting list of tasks: %v\n", err)
		return
	}

	for _, t := range tasks.([]*task.Task) {
		if t.State == task.Running {
			resp := w.InspectTask(*t)
			if resp.Error != nil {
				fmt.Printf("ERROR: %v\n", resp.Error)
				continue
			}

			if resp.Container == nil {
				log.Printf("No container for running task %s\n", t.ID)
				w.Db.Put(t.ID.String(), t)
				continue
			}

			if resp.Container.State.Status == "exited" {
				log.Printf("Container for task %s in non-running state %s",
					t.ID, resp.Container.State.Status)
				w.Db.Put(t.ID.String(), t)
				continue
			}

			t.HostPorts = resp.Container.NetworkSettings.Ports
			w.Db.Put(t.ID.String(), t)
		}
	}
}