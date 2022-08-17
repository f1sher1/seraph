package config

import (
	"fmt"

	"github.com/go-ini/ini"
)

type DatabaseConfig struct {
	Connection  string `json:"connection"`
	Debug       bool   `json:"debug"`
	PoolSize    int    `json:"pool_size"`
	IdleTimeout int    `json:"idle_timeout"`
}

func NewDefaultDatabaseConfig(c *ini.Section) DatabaseConfig {
	host := c.Key("host").String()
	port := c.Key("port").Value()
	user := c.Key("user").Value()
	passwd := c.Key("passwd").Value()
	debug, _ := c.Key("debug").Bool()
	pool_size, _ := c.Key("pool_size").Int()
	idle_timeout, _ := c.Key("idle_timeout").Int()
	return DatabaseConfig{
		Connection:  fmt.Sprintf("mysql://%s:%s@%s:%s/seraph?charset=utf8&parseTime=True&loc=Local", user, passwd, host, port),
		Debug:       debug,
		PoolSize:    pool_size,
		IdleTimeout: idle_timeout,
	}
}
