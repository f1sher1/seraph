package driver

import (
	"fmt"
	"net/url"
	"seraph/pkg/oslo-messaging-go/driver/rabbit"
	"seraph/pkg/oslo-messaging-go/interfaces"
)

func Open(urlStr string) (interfaces.Dialector, error) {
	uri, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	switch uri.Scheme {
	case "rabbit":
		return rabbit.Open(urlStr), nil
	default:
		return nil, fmt.Errorf("unsupported schema %s", uri.Scheme)
	}
}
