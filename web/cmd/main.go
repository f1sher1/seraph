package main

import (
	"net/http"
	"seraph/app/config"
	"seraph/app/db"
	"seraph/pkg/log"
	"seraph/web/handles"
	"time"

	"github.com/julienschmidt/httprouter"
)

func init() {
	timelocal := time.FixedZone("CST", 3600*8)
	time.Local = timelocal
	log.Info(nil, "Start running web server...")
	cfg := &db.Config{
		Connection:  config.Config.Database.Connection,
		Debug:       config.Config.Database.Debug,
		PoolSize:    config.Config.Database.PoolSize,
		IdleTimeout: config.Config.Database.IdleTimeout,
	}
	if err := db.Init(cfg); err != nil {
		log.Error(nil, err)
		panic(err)
	}

}
func main() {

	router := httprouter.New()

	router.POST("/:tenantid/servers/:uuid/action", handles.BaseHandles)

	log.Error(nil, http.ListenAndServe(":8080", router))
}
