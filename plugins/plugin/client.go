package plugin

import (
	"seraph/pkg/pluginx"
	"sync"
)

var (
	client = NewClient()
)

func NewClient() *Client {
	return &Client{clients: map[string]*pluginx.PluginClient{}}
}

type Client struct {
	mu      sync.Mutex
	clients map[string]*pluginx.PluginClient
}

func (c *Client) getClient(name string) *pluginx.PluginClient {
	c.mu.Lock()
	defer c.mu.Unlock()

	sockAddr := GetSockPath(name)
	if clt, ok := c.clients[sockAddr]; ok {
		return clt
	} else {
		clt = pluginx.NewPluginClient(sockAddr)
		c.clients[sockAddr] = clt
		return clt
	}
}

func (c *Client) Call(name string, attrs map[string]interface{}, args interface{}) (interface{}, error) {
	clt := c.getClient(name)
	ctx, err := clt.Prepare(attrs)
	if err != nil {
		return nil, err
	}
	return ctx.Call(name, args)
}

func Call(name string, attrs map[string]interface{}, args interface{}) (interface{}, error) {
	return client.Call(name, attrs, args)
}
