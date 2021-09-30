package dbus

//dbus-send --system --type=method_call --dest=org.freedesktop.login1 /org/freedesktop/login1 org.freedesktop.login1.Manager.ScheduleShutdown string:"reboot" uint64:1632282493000000

import (
	"fmt"
	"os"

	"github.com/Fishwaldo/go-logadapter"
	"github.com/godbus/dbus/v5"
)

type DbusS struct {
}

var DBUS DbusS

func (d DbusS) Start(logadapter.Logger) {
	var list []string
	conn, err := dbus.SystemBus()
	if err != nil {
		panic(err)
	}

	err = conn.BusObject().Call("org.freedesktop.DBus.ListNames", 0).Store(&list)
	if err != nil {
		panic(err)
	}
	for _, v := range list {
		fmt.Println(v)
	}
	var s string
	err = conn.Object("org.freedesktop.login1", "/").Call("org.freedesktop.DBus.Introspectable.Introspect", 0).Store(&s)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to introspect bluez", err)
		os.Exit(1)
	}

	fmt.Println(s)
}
