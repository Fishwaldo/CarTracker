package taskmanager

import (
	"github.com/go-logr/logr"
	"github.com/Fishwaldo/go-taskmanager"

	//"github.com/Fishwaldo/go-taskmanager/job"
	//executionmiddleware "github.com/Fishwaldo/go-taskmanager/middleware/executation"
	//retrymiddleware "github.com/Fishwaldo/go-taskmanager/middleware/retry"
)

//var logger logadapter.Logger
var scheduler *taskmanager.Scheduler

func InitScheduler(log logr.Logger) {
	scheduler = taskmanager.NewScheduler(
		taskmanager.WithLogger(log),
	)
}

func StartScheduler() (bool) {
	scheduler.StartAll()
	return true;
}

func GetScheduler() (*taskmanager.Scheduler) {
	return scheduler
}

func StopScheduler() {
	scheduler.StopAll()
}
