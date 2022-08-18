package service

import (
	messaging "seraph/pkg/oslo-messaging-go"
	"seraph/pkg/oslo-messaging-go/driver"
	"seraph/pkg/oslo-messaging-go/message"
	"sync"
)

var (
	mu             sync.Mutex
	_transportsMap = map[string]*messaging.Transport{}
)

func GetTransport(connStr string) (*messaging.Transport, error) {
	mu.Lock()
	defer mu.Unlock()

	if trans, ok := _transportsMap[connStr]; ok {
		return trans, nil
	} else {
		dialector, err := driver.Open(connStr)
		if err != nil {
			return nil, err
		}

		_trans, err := messaging.NewTransport(dialector)
		if err != nil {
			return nil, err
		}
		_transportsMap[connStr] = _trans
		return _trans, nil
	}
}

type RPCServiceConfig struct {
	DriverStr  string
	Exchange   string
	Host       string
	Topic      string
	Version    string
	Properties map[string]interface{}
}

func NewRPCServiceConfig(driverStr string, exchange string, host string, topic string, version string, properties map[string]interface{}) *RPCServiceConfig {
	return &RPCServiceConfig{
		DriverStr:  driverStr,
		Exchange:   exchange,
		Host:       host,
		Topic:      topic,
		Version:    version,
		Properties: properties,
	}
}

type RPCService struct {
}

func (r *RPCService) InitializeRPCServer(cfg *RPCServiceConfig) (*messaging.RPCServer, error) {
	transport, err := GetTransport(cfg.DriverStr)
	if err != nil {
		return nil, err
	}

	target := message.Target{
		Exchange: cfg.Exchange,
		Topic:    cfg.Topic,
		Host:     cfg.Host,
		Version:  cfg.Version,
	}

	dispatcher := messaging.NewRPCDispatcher()
	return messaging.NewRPCServer(transport, target, &dispatcher), nil
}
