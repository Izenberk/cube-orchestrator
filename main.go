package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/Izenberk/cube-orchestrator/manager"
	"github.com/Izenberk/cube-orchestrator/worker"
	"github.com/joho/godotenv"
)

func main() {

	_ = godotenv.Load()

	whost := os.Getenv("CUBE_WORKER_HOST")
	wport, _ := strconv.Atoi(os.Getenv("CUBE_WORKER_PORT"))

	mhost := os.Getenv("CUBE_MANAGER_HOST")
	mport, _ := strconv.Atoi(os.Getenv("CUBE_MANAGER_PORT"))

	fmt.Println("Starting Cube worker")

	// w1 := worker.New("worker-1", "memory")
	w1 := worker.New("worker-1", "persistent")
	wapi1 := worker.Api{Address: whost, Port: wport, Worker: w1}

	// w2 := worker.New("worker-2", "memory")
	w2 := worker.New("worker-2", "persistent")
	wapi2 := worker.Api{Address: whost, Port: wport + 1, Worker: w2}

	// w3 := worker.New("worker-3", "memory")
	w3 := worker.New("worker-3", "persistent")
	wapi3 := worker.Api{Address: whost, Port: wport + 2, Worker: w3}

	go w1.RunTasks()
	go w1.UpdateTasks()
	go wapi1.Start()

	go w2.RunTasks()
	go w2.UpdateTasks()
	go wapi2.Start()

	go w3.RunTasks()
	go w3.UpdateTasks()
	go wapi3.Start()

	// 💡 Allow HTTP servers time to bind sockets before Manager polls
	time.Sleep(100 * time.Millisecond)

	fmt.Println("Starting Cube manager")

	workers := []string{
		fmt.Sprintf("%s:%d", whost, wport),
		fmt.Sprintf("%s:%d", whost, wport+1),
		fmt.Sprintf("%s:%d", whost, wport+2),
	}
	// m := manager.New(workers, "epvm", "memory")
	m := manager.New(workers, "epvm", "persistent")
	mapi := manager.Api{Address: mhost, Port: mport, Manager: m}

	go m.ProcessTasks()
	go m.UpdateTasks()
	go m.DoHealthChecks()

	mapi.Start()
}

