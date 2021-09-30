package internal

import (
	"context"
	"github.com/Fishwaldo/go-logadapter"
)

type PluginI interface {
	Start(logadapter.Logger)
	Stop()
	Poll(context.Context)
}

var Plugins map[string]PluginI

func init() {
	Plugins = make(map[string]PluginI)
}

var log logadapter.Logger

func RegisterPlugin(name string, pi PluginI) {
	_, ok := Plugins[name]; 
	if ok {
		return
	}
	Plugins[name] = pi
}

func StartPlugins(logger logadapter.Logger) {
	log = logger
	for name, pi := range Plugins {
		log.Info("Starting Plugin %s", name)
		pi.Start(log.New(name))
	}
}

func StopPlugins() {
	for name, pi := range Plugins {
		log.Info("Stopping Plugin %s", name)
		pi.Stop()
	}
}