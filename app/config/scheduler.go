package config

type SchedulerConfig struct {
	Kind  string  `json:"kind"`
	Delay float64 `json:"delay"`
}

func NewDefaultSchedulerConfig() SchedulerConfig {
	return SchedulerConfig{
		Kind:  "default",
		Delay: 1000,
	}
}
