package manager

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Izenberk/cube-orchestrator/task"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)


func (a *Api) StartTaskHandler(w http.ResponseWriter, r *http.Request) {
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()

	te := task.TaskEvent{}
	err := d.Decode(&te)
	if err != nil {
		msg := fmt.Sprintf("Error unmarshalling body: %v\n", err)
		log.Printf("%s", msg)
		w.WriteHeader(http.StatusBadRequest)
		e := ErrResponse {
			HTTPStatusCode: http.StatusBadRequest,
			Message: 				msg,
		}
		json.NewEncoder(w).Encode(e)
		return
	}

	a.Manager.AddTask(te)
	log.Printf("Added task %v\n", te.Task.ID)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(te.Task)
}

func (m *Manager) GetTasks() []*task.Task {
	tasks := []*task.Task{}
	for _, t := range m.TaskDb {
		tasks = append(tasks, t)
	}
	return tasks
}

func (a *Api) GetTasksHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(a.Manager.GetTasks())
}

func (a *Api) StopTaskHandler(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")
	if taskID == "" {
		log.Printf("No taskID passed in request.\n")
		w.WriteHeader(http.StatusBadRequest)
	}

	tID, _ := uuid.Parse(taskID)
	taskToStop, ok := a.Manager.TaskDb[tID]
	if !ok {
		log.Printf("No task with ID %v found", tID)
		w.WriteHeader(http.StatusNotFound)
	}

	te := task.TaskEvent{
		ID: 				uuid.New(),
		State: 			task.Completed,
		Timestamp: 	time.Now(),
	}
	taskCopy := *taskToStop
	taskCopy.State = task.Completed
	te.Task = taskCopy
	a.Manager.AddTask(te)

	log.Printf("Added task event %v to stop %v\n", te.ID, taskToStop.ID)
	w.WriteHeader(http.StatusNoContent)
}