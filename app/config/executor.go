package config

import (
	"os"

	"github.com/go-ini/ini"
)

type ExecutorConfig struct {
	Kind  string `json:"kind"`
	Host  string `json:"host"`
	Topic string `json:"topic"`
}

func NewDefaultExecutorConfig(c *ini.Section) ExecutorConfig {
	myip := c.Key("myip").Value()
	if myip == "" {
		myip = os.Getenv("HOSTNAME")
	}
	return ExecutorConfig{
		Kind:  "local",
		Host:  myip,
		Topic: "seraph-executor",
	}
}
