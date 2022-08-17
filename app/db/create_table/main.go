package main

import (
	"fmt"
	"seraph/app/config"
	"seraph/app/db"
)

func main() {

	fmt.Println(config.Config.Database.Connection)
	cfg := &db.Config{
		Connection:  config.Config.Database.Connection,
		Debug:       config.Config.Database.Debug,
		PoolSize:    config.Config.Database.PoolSize,
		IdleTimeout: config.Config.Database.IdleTimeout,
	}
	if err := db.Init(cfg); err != nil {
		panic(err)
	}
	if err := db.Migrate(); err != nil {
		panic(err)
	}
	fmt.Println("Create tables over!")
}
