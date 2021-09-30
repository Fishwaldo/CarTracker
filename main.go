package main

import (
	//"context"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"github.com/Fishwaldo/CarTracker/internal"
	_ "github.com/Fishwaldo/CarTracker/internal/config"
	//"github.com/Fishwaldo/CarTracker/internal/dbus"
	_ "github.com/Fishwaldo/CarTracker/internal/gps"
	"github.com/Fishwaldo/CarTracker/internal/natsconnection"
	_ "github.com/Fishwaldo/CarTracker/internal/perf"
	_ "github.com/Fishwaldo/CarTracker/internal/powersupply"
	"github.com/Fishwaldo/CarTracker/internal/taskmanager"
	"github.com/Fishwaldo/CarTracker/internal/update"
	"github.com/Fishwaldo/CarTracker/internal/web"
	"github.com/Fishwaldo/go-logadapter/loggers/logrus"
)

var (
    version = "0.0.1"
    commit  = "none"
    date    = "unknown"
    builtBy = "unknown"
)
func init() {
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Sum != "" {
		fmt.Println(info)
	  }
}

func main() {

	fmt.Printf("Starting CarTracker %s - %s (built by %s on %s)\n", version, commit, builtBy, date)
	logger := logrus.LogrusDefaultLogger()

	//dbus.DBUS.Start(logger)

	err := update.DoUpdate(version)
	if err != nil {
		fmt.Printf("Error: %s", err)
	}
	
	return

	//logger.SetLevel(logadapter.LOG_TRACE)
	natsconnection.Nats.Start(logger.New("NATS"))
	web.Web.Start(logger.New("WEB"))
	taskmanager.InitScheduler(logger.New("TaskManager"))

	internal.StartPlugins(logger)

	taskmanager.StartScheduler()

	logger.Info("Server Started")

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	s := <-signalChan
	logger.Info("Got Shutdown Signal %s", s)

	internal.StopPlugins()

	taskmanager.StopScheduler()

}
