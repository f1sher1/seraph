package pluginx

import (
	"net/rpc"
	"net/rpc/jsonrpc"
	"net/url"
)

type PluginClientContext struct {
	client *rpc.Client
	attrs  map[string]interface{}
}

func (ctx *PluginClientContext) Call(name string, args interface{}) (interface{}, error) {
	reply := MessageReply{}
	msg := MessageArgs{
		Attributes: ctx.attrs,
		Message:    args,
	}
	err := ctx.client.Call(name+".Run", &msg, &reply)
	if err != nil {
		return nil, err
	}
	return reply.Message, nil
}

type PluginClient struct {
	addr string
}

func (c *PluginClient) Prepare(attrs map[string]interface{}) (*PluginClientContext, error) {
	uri, err := url.Parse(c.addr)
	if err != nil {
		return nil, err
	}

	client, err := jsonrpc.Dial(uri.Scheme, uri.Host+uri.Path)
	if err != nil {
		return nil, err
	}
	return &PluginClientContext{
		client: client,
		attrs:  attrs,
	}, err
}

func NewPluginClient(addr string) *PluginClient {
	return &PluginClient{addr: addr}
}
