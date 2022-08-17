package config

import (
	"os"

	"github.com/go-ini/ini"
)

type EngineConfig struct {
	Host  string `json:"host"`
	Topic string `json:"topic"`
}

func NewDefaultEngineConfig(c *ini.Section) EngineConfig {
	myip := c.Key("myip").Value()
	if myip == "" {
		myip = os.Getenv("HOSTNAME")
	}
	return EngineConfig{
		Host:  myip,
		Topic: "seraph-engine",
	}
}
