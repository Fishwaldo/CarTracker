package taskmanager

import (
	"github.com/Fishwaldo/go-logadapter"
	"github.com/Fishwaldo/go-taskmanager"
	//"github.com/Fishwaldo/go-taskmanager/job"
	//executionmiddleware "github.com/Fishwaldo/go-taskmanager/middleware/executation"
	//retrymiddleware "github.com/Fishwaldo/go-taskmanager/middleware/retry"
	logruslog "github.com/Fishwaldo/go-taskmanager/loggers/logrus"
)

//var logger logadapter.Logger
var scheduler *taskmanager.Scheduler

func InitScheduler(log logadapter.Logger) {
	logger :=  logruslog.LogrusDefaultLogger();
	scheduler = taskmanager.NewScheduler(
		taskmanager.WithLogger(logger),
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
