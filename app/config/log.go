package config

import (
	"runtime"

	"github.com/go-ini/ini"
)

type LogConfig struct {
	Format          string `json:"format"`
	TimestampFormat string `json:"timestamp_format"`
	DirPath         string `json:"dir_path"`
}

func NewDefaultLogConfig(c *ini.Section) LogConfig {
	dir_path := c.Key("dir_path").String()
	if dir_path == "" {
		sysType := runtime.GOOS
		if sysType == "windows" {
			dir_path = "C:\\log"
		} else {
			dir_path = "/var/log"
		}
	}

	return LogConfig{
		Format:          "{{.timestamp}} {{.pid}} [{{.name}}] [{{.levelname}}] [{{.requestId}} {{.workflow}}] {{.message}}",
		TimestampFormat: "2006-01-02 15:04:05.000",
		DirPath:         dir_path,
	}
}
