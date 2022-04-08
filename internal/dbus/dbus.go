package dbus

//dbus-send --system --type=method_call --dest=org.freedesktop.login1 /org/freedesktop/login1 org.freedesktop.login1.Manager.ScheduleShutdown string:"reboot" uint64:1632282493000000

import (

	//	"time"
	"time"

	dcdcusb "github.com/Fishwaldo/go-dcdc200"
	"github.com/sasha-s/go-deadlock"
	"github.com/go-logr/logr"
	"github.com/godbus/dbus/v5"
)

type DbusS struct {
	log      logr.Logger
	mx       deadlock.RWMutex
	conn     *dbus.Conn
	curstate dcdcusb.DcdcStatet
}

var DBUS *DbusS

func (d *DbusS) Start(log logr.Logger) {
	d = &DbusS{}
	d.log = log
	var list []string
	var err error
	d.conn, err = dbus.SystemBus()
	if err != nil {
		d.log.Error(err, "Dbus Connection Error")
	}

	err = d.conn.BusObject().Call("org.freedesktop.DBus.ListNames", 0).Store(&list)
	if err != nil {
		d.log.Error(err, "Dbus Call Error")
	}
	for _, v := range list {
		d.log.Info("Info: ", "list", v)
	}
	DBUS = d
}

func (d *DbusS) ProcessPowerSupply(param dcdcusb.Params) error {
	d.mx.Lock()
	defer d.mx.Unlock()
	if d.curstate != param.State {
		d.log.Info("Power State Transition", "state", param.State)
		d.curstate = param.State
		if d.curstate == dcdcusb.StateIgnOff {
			d.log.Info("Ignition Powered Off - Scheduling Shutdown")
		}
	}
	if d.curstate == dcdcusb.StateIgnOff {
		if param.State == dcdcusb.StateOk {
			d.log.Info("Ignition On - Canceling Shutdown")
			d.curstate = param.State
		} else {
			d.log.Info("Shutdown in", "timeout", param.TimerSoftOff.Seconds())
			if param.TimerSoftOff.Seconds() < 60 {
				when := time.Now().Add(time.Minute)
				d.log.Info("Shutdown at ", "when", when.String())
				call := d.conn.Object("org.freedesktop.login1", "/org/freedesktop/login1").Call("org.freedesktop.login1.Manager.ScheduleShutdown", 0, "dry-poweroff", uint64(when.UnixMicro()))
				if call.Err != nil {
					d.log.Error(call.Err, "Failed to Schedule Shutdown")
				}			
			}
		}
	}
	return nil
}
