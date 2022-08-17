package config

import "github.com/go-ini/ini"

var (
	// LoadFile, _ = ini.Load("config/config.ini")
	LoadFile, _ = ini.Load("/opt/config.ini")
	Config      = Configuration{
		API:         NewDefaultAPIConfig(),
		Database:    NewDefaultDatabaseConfig(LoadFile.Section("db")),
		Engine:      NewDefaultEngineConfig(LoadFile.Section("myip")),
		Executor:    NewDefaultExecutorConfig(LoadFile.Section("myip")),
		Scheduler:   NewDefaultSchedulerConfig(),
		Messaging:   NewMessagingConfig(LoadFile.Section("rabbitMQ")),
		LOG:         NewDefaultLogConfig(LoadFile.Section("log")),
		NovaClient:  NewNovaConfig(LoadFile.Section("nova")),
		UniMQClient: NewUniMQConfig(LoadFile.Section("uniMQ")),
	}
)

type Configuration struct {
	API         APIConfig       `json:"api"`
	Database    DatabaseConfig  `json:"database"`
	Engine      EngineConfig    `json:"engine"`
	Executor    ExecutorConfig  `json:"executor"`
	Scheduler   SchedulerConfig `json:"scheduler"`
	Messaging   MessagingConfig `json:"messaging"`
	LOG         LogConfig       `json:"log"`
	NovaClient  NovaConfig      `json:"nova"`
	UniMQClient UniMQConfig     `json:"unimq"`
}

func (c *Configuration) Initialize(configFile string) error {
	return nil
}

func Initialize(configFile string) error {
	return Config.Initialize(configFile)
}
