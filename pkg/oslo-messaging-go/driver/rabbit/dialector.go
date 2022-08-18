package rabbit

import (
	"seraph/pkg/oslo-messaging-go/interfaces"
)

func Open(url string) interfaces.Dialector {
	return &Dialector{Config: &Config{Url: url}}
}

type Config struct {
	DriverName string
	Url        string

	Conn interfaces.ConnPool
}

type Dialector struct {
	*Config
}

func (dialector Dialector) Name() string {
	return "rabbit"
}

func (dialector Dialector) Initialize(transport interfaces.Transport) error {
	if dialector.DriverName == "" {
		dialector.DriverName = "rabbit"
	}

	connector, err := NewConnector(dialector.Url)
	if err != nil {
		return err
	}
	transport.SetConnPool(NewPool(*connector))

	return nil
}
