package config

type APIConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

func NewDefaultAPIConfig() APIConfig {
	return APIConfig{
		Host: "0.0.0.0",
		Port: 8791,
	}
}
