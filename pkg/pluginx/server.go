package pluginx

import (
	"fmt"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"net/url"
	"os"
	"reflect"
	"seraph/pkg/log"
)

const (
	endpointRunFunc = "Run"
)

func callEndpoint(callback EndpointCallable, arguments interface{}) (interface{}, error) {
	cb := reflect.ValueOf(callback)
	_method, ok := cb.Type().MethodByName(endpointRunFunc)
	if !ok {
		return nil, fmt.Errorf("invalid endpoint, there isn't a function called %s", endpointRunFunc)
	}

	methodType := _method.Type
	if methodType.NumIn() != 2 {
		return nil, fmt.Errorf("invalid endpoint, the arguments num must be 2")
	}

	argType := methodType.In(1)

	callArg, err := Deserialize(arguments, argType)
	if err != nil {
		return nil, err
	}

	method := cb.MethodByName(endpointRunFunc)
	results := method.Call([]reflect.Value{
		reflect.ValueOf(callArg),
	})

	replyErr := results[1].Interface()
	if replyErr == nil {
		return Serialize(results[0].Interface())
	} else {
		return nil, replyErr.(error)
	}
}

type Endpoint struct {
	callback EndpointCallable
}

func (e *Endpoint) Run(args *MessageArgs, reply *MessageReply) error {
	var callback EndpointCallable
	callback = e.callback.Initialize(args.Attributes)

	result, err := callEndpoint(callback, args.Message)
	if err != nil {
		return err
	}
	reply.Message = result
	return nil
}

type EndpointCallable interface {
	Initialize(map[string]interface{}) EndpointCallable
}

type PluginServer struct {
	server   *rpc.Server
	listener net.Listener
}

func (s *PluginServer) Initialize() error {
	s.server = rpc.NewServer()
	return nil
}

func (s *PluginServer) Register(name string, endpoint EndpointCallable) error {
	return s.server.RegisterName(name, &Endpoint{endpoint})
}

func (s *PluginServer) Serve() error {
	for {
		log.Debug(nil, "Get new rpc request in server")
		conn, err := s.listener.Accept()
		if err != nil {
			continue
		}
		go s.server.ServeCodec(jsonrpc.NewServerCodec(conn))
	}
}

func (s *PluginServer) Shutdown() error {
	return s.listener.Close()
}

func NewPluginServer(addr string) (*PluginServer, error) {
	uri, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}

	if uri.Scheme == "unix" {
		path := uri.Host + uri.Path
		_, err = os.Stat(path)
		if err == nil {
			_ = os.Remove(path)
		}
	}

	l, err := net.Listen(uri.Scheme, uri.Host+uri.Path)
	if err != nil {
		return nil, err
	}
	server := &PluginServer{
		listener: l,
	}
	err = server.Initialize()
	if err != nil {
		return nil, err
	}
	return server, nil
}
