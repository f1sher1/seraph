package main

import (
	"os"
	"os/signal"
	"seraph/plugins/nova"
	"seraph/plugins/plugin"
	"syscall"
	"time"
)

func init() {
	timelocal := time.FixedZone("CST", 3600*8)
	time.Local = timelocal
}

func main() {
	server, err := nova.NewPluginServer(plugin.GetSockPath("nova"))
	if err != nil {
		panic(err)
	}
	go server.Serve()
	sign := make(chan os.Signal, 1)
	signal.Notify(sign, os.Interrupt, syscall.SIGTERM)
	<-sign
	server.Shutdown()
}
