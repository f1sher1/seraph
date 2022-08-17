package standard

import (
	"seraph/pkg/log"
	"seraph/pkg/pluginx"
)

type StdEcho struct {
	attributes map[string]interface{}
}

type StdEchoInput struct {
	Output string `json:"output"`
}

func (s *StdEcho) Initialize(attrs map[string]interface{}) pluginx.EndpointCallable {
	return &StdEcho{attributes: attrs}
}

func (s *StdEcho) Run(input StdEchoInput) (string, error) {
	log.Debugf(nil, "run std echo input is %#v", input)
	return input.Output, nil
}

type StdTestInput struct {
	Name   string            `json:"name"`
	Age    float64           `json:"age"`
	School map[string]string `json:"school"`
}

type StdTestOutput struct {
	StdTestInput
	Replied bool `json:"replied"`
}

type StdTest struct {
	attributes map[string]interface{}
}

func (s *StdTest) Initialize(attrs map[string]interface{}) pluginx.EndpointCallable {
	return &StdTest{attributes: attrs}
}

func (s *StdTest) Run(input StdTestInput) (interface{}, error) {
	log.Debugf(nil, "got input %v", input)
	return StdTestOutput{
		StdTestInput: input,
		Replied:      true,
	}, nil
}

var (
	endpoints = map[string]pluginx.EndpointCallable{
		"std.echo": &StdEcho{},
		"std.test": &StdTest{},
	}
)

func NewPluginServer(addr string) (*pluginx.PluginServer, error) {
	server, err := pluginx.NewPluginServer(addr)
	if err != nil {
		return nil, err
	}
	for name, endpoint := range endpoints {
		err = server.Register(name, endpoint)
		if err != nil {
			return nil, err
		}
	}
	return server, nil
}

func GetEndpoints() map[string]pluginx.EndpointCallable {
	return endpoints
}
