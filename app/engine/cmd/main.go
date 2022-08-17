package main

import (
	"os"
	"os/signal"
	"seraph/app/config"
	"seraph/app/db"
	"seraph/app/engine/server"
	"syscall"
	"time"
)

func init() {
	timelocal := time.FixedZone("CST", 3600*8)
	time.Local = timelocal
}

func main() {
	config.Initialize("")
	dbCfg := config.Config.Database
	cfg := &db.Config{
		Connection:  dbCfg.Connection,
		Debug:       dbCfg.Debug,
		PoolSize:    dbCfg.PoolSize,
		IdleTimeout: dbCfg.IdleTimeout,
	}
	err := db.Init(cfg)
	if err != nil {
		panic(err)
	}

	svc := server.NewEngineServer()
	if err := svc.Initialize(); err != nil {
		panic(err)
	}
	go svc.Start()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	<-sigs
	svc.Stop()
}
