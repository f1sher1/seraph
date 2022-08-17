package config

import (
	"fmt"

	"github.com/go-ini/ini"
)

type MessagingConfig struct {
	Connection string `json:"connection"`
	Exchange   string `json:"exchange"`
	Version    string `json:"version"`
}

func NewMessagingConfig(c *ini.Section) MessagingConfig {
	host := c.Key("host").Value()
	user := c.Key("user").Value()
	passwd := c.Key("passwd").Value()
	amqp_queues_ttl := c.Key("amqp_queues_ttl").Value()
	exchange := c.Key("exchange").Value()
	return MessagingConfig{
		Connection: fmt.Sprintf("rabbit://%s:%s@%s/?amqp_queues_ttl=%s&amqp_ha_queues=true", user, passwd, host, amqp_queues_ttl),
		Exchange:   exchange,
		Version:    "2.0",
	}
}
