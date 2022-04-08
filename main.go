package main

import (
	//"context"
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"github.com/Fishwaldo/CarTracker/internal"
	_ "github.com/Fishwaldo/CarTracker/internal/config"

	"github.com/Fishwaldo/CarTracker/internal/dbus"
	_ "github.com/Fishwaldo/CarTracker/internal/gps"
	"github.com/Fishwaldo/CarTracker/internal/natsconnection"
	_ "github.com/Fishwaldo/CarTracker/internal/perf"
	_ "github.com/Fishwaldo/CarTracker/internal/powersupply"
	"github.com/Fishwaldo/CarTracker/internal/taskmanager"
	"github.com/Fishwaldo/CarTracker/internal/update"
	"github.com/Fishwaldo/CarTracker/internal/web"
	"github.com/blang/semver/v4"
	"github.com/bombsimon/logrusr/v2"
	"github.com/sirupsen/logrus"
)

var (
    VersionSummary = "0.0.0"
)
func init() {
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Sum != "" {
		fmt.Println(info)
	  }
}

func main() {
	logsink := logrus.New()
	logger := logrusr.New(logsink)

	//logger.SetLevel(logadapter.LOG_TRACE)


	version, err := semver.ParseTolerant(VersionSummary)
	if err != nil {
		version, _ = semver.Make("0.0.0")
	}
	/* construct a version */
	versionstring := version.FinalizeVersion()
	if len(version.Pre) > 0 {
		versionstring = fmt.Sprintf("%s-%s", versionstring, version.Pre[0].VersionStr)
		if len(version.Build) > 0 {
			versionstring = fmt.Sprintf("%s-%s", versionstring, version.Build[0])
		}
	}
	
	logger.Info("Starting CarTracker", "version", versionstring)
	if latest, err := update.GitHubLatestRelease(context.Background(), "Fishwaldo", "CarTracker"); err != nil {
		logger.Error(err, "Cant Find Latest Release Info From Github")
	} else {
		logger.Info("Latest Release", "version", latest.String());
	}

	dbus.DBUS.Start(logger)

	natsconnection.Nats.Start(logger.WithName("NATS"))
	web.Web.Start(logger.WithName("WEB"))
	taskmanager.InitScheduler(logger.WithName("TaskManager"))

	internal.StartPlugins(logger)

	internal.InitStatusUpdate()

	taskmanager.StartScheduler()

	logger.Info("Server Started")

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	s := <-signalChan
	logger.Info("Got Shutdown Signal", "signal", s)

	internal.StopPlugins()

	taskmanager.StopScheduler()

}
