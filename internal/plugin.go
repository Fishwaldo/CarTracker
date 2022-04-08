package internal

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Fishwaldo/CarTracker/internal/dbus"
	"github.com/Fishwaldo/CarTracker/internal/natsconnection"
	"github.com/Fishwaldo/CarTracker/internal/web"
	tm "github.com/Fishwaldo/CarTracker/internal/taskmanager"
	"github.com/Fishwaldo/go-taskmanager"
	dcdcusb "github.com/Fishwaldo/go-dcdc200"
	"github.com/go-logr/logr"
)
type power struct {
	Vin float32
	Vout float32
	PowerOn bool
	ShutDownTimeout time.Duration	
}

type StatusMsg struct { 
	Power power
}

var State StatusMsg


type PluginI interface {
	Start(logr.Logger)
	Stop()
	Poll(context.Context)
}

var Plugins map[string]PluginI

func init() {
	Plugins = make(map[string]PluginI)
}

var log logr.Logger

func RegisterPlugin(name string, pi PluginI) {
	_, ok := Plugins[name]; 
	if ok {
		return
	}
	Plugins[name] = pi
}

func StartPlugins(logger logr.Logger) {
	log = logger
	for name, pi := range Plugins {
		log.Info("Starting Plugin", "plugin", name)
		pi.Start(log.WithName(name))
	}
}

func StopPlugins() {
	for name, pi := range Plugins {
		log.Info("Stopping Plugin", "plugin", name)
		pi.Stop()
	}
}

func ProcessUpdate(plugin string, data interface{}) error {
	natsconnection.Nats.SendStats(plugin, data)

	switch msg := data.(type) {
	case dcdcusb.Params:
		dbus.DBUS.ProcessPowerSupply(msg)
		State.Power.Vin = msg.Vin
		State.Power.Vout = msg.VoutActual
		if msg.State == dcdcusb.StateOk {
			State.Power.PowerOn = true
		} else { 
			State.Power.PowerOn = false
		}
	}
	return nil;
}

func InitStatusUpdate() {
	fixedTimer1second, err := taskmanager.NewFixed(1 * time.Second)
	if err != nil {
		log.Error(err, "invalid interval")
	}

	err = tm.GetScheduler().Add(context.TODO(), "StatusBroadcast", fixedTimer1second, SendPeriodUpdate)
	if err != nil {
		log.Error(err, "Can't Initilize Scheduler for StatusBroadcast")
	}
	log.Info("Initilized Scheduler for StatusBroadcast")
}

func SendPeriodUpdate(ctx context.Context) {
	log.Info("Sending Status Update")
	if txt, err := json.Marshal(State); err != nil {
		log.Error(err, "Error marshalling state")
	} else {
		web.Web.Broadcast(string(txt))
	}
}