package worker

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/Izenberk/cube-orchestrator/task"
	"github.com/go-chi/chi/v5"
)

type Api struct {
	Address 		string
	Port 				int
	Worker			*Worker
	Router			*chi.Mux
}

type ErrResponse struct {
	HTTPStatusCode		int 		`json:"http_status_code"`
	Message 					string 	`json:"message"`
}

func (a *Api) StartTaskHandler(w http.ResponseWriter, r *http.Request) {
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()

	te := task.TaskEvent{}
	err := d.Decode(&te)
	if err != nil {
		msg := fmt.Sprintf("Error unmarshalling body: %v\n", err)
		log.Printf("%s", msg)
		w.WriteHeader(400)
		e := ErrResponse{
			HTTPStatusCode: 	400,
			Message:					msg,
		}
		json.NewEncoder(w).Encode(e)
		return
	}

	a.Worker.AddTask(te.Task)
	log.Printf("Added task %v\n", te.Task.ID)
	w.WriteHeader(201)
	json.NewEncoder(w).Encode(te.Task)
}

func GetTasksHandler(w http.ResponseWriter, r *http.Request) {}

func StopTaskHandler(w http.ResponseWriter, r *http.Request) {}

func (a *Api) GetTasksHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(a.Worker.GetTasks())
}