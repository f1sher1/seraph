package main

import (
	"os"
	"os/signal"
	"seraph/plugins/plugin"
	"seraph/plugins/standard"
	"syscall"
)

func main() {
	server, err := standard.NewPluginServer(plugin.GetSockPath("std"))
	if err != nil {
		panic(err)
	}
	go server.Serve()
	sign := make(chan os.Signal, 1)
	signal.Notify(sign, os.Interrupt, syscall.SIGTERM)
	<-sign
	server.Shutdown()
}
