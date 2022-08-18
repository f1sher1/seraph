package config

import "github.com/go-ini/ini"

type NovaConfig struct {
	URL string `json:"url"`
}

func NewNovaConfig(c *ini.Section) NovaConfig {
	url := c.Key("url").Value()
	return NovaConfig{
		URL: url,
	}
}

type UniMQConfig struct {
	URL        string `json:"url"`
	TopicName  string `json:"topic_name"`
	AppKey     string `json:"app_key"`
	RoutingKey string `json:"routing_key"`
	SecretKey  string `json:"secret_key"`
}

func NewUniMQConfig(c *ini.Section) UniMQConfig {
	url := c.Key("url").Value()
	topic_name := c.Key("topic_name").Value()
	routing_key := c.Key("routing_key").Value()
	app_key := c.Key("app_key").Value()
	secret_key := c.Key("secret_key").Value()
	return UniMQConfig{
		URL:        url,
		TopicName:  topic_name,
		AppKey:     app_key,
		RoutingKey: routing_key,
		SecretKey:  secret_key,
	}
}
