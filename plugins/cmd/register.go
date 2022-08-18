package main

import (
	"seraph/app/config"
	"seraph/app/db"
	"seraph/pkg/log"
	"seraph/plugins"
)

func main() {
	config.Initialize("")
	log.Initialize(config.Config.LOG.Format, config.Config.LOG.TimestampFormat)
	db.Init(&db.Config{
		Connection:  config.Config.Database.Connection,
		Debug:       config.Config.Database.Debug,
		PoolSize:    config.Config.Database.PoolSize,
		IdleTimeout: config.Config.Database.IdleTimeout,
	})

	err := plugins.RegisterBuiltinActions()
	if err != nil {

	}
}
