package messaging

import (
	"context"
	"errors"
	"reflect"
	"seraph/pkg/contextx"
	"seraph/pkg/log"
	"seraph/pkg/oslo-messaging-go/interfaces"
	"seraph/pkg/oslo-messaging-go/message"
	"time"
)

type RPCServer struct {
	transport    *Transport
	target       message.Target
	dispatcher   interfaces.Dispatcher
	listener     interfaces.Listener
	stopListener context.CancelFunc
}

func NewRPCServer(transport *Transport, target message.Target, dispatcher interfaces.Dispatcher) *RPCServer {
	return &RPCServer{
		transport:  transport,
		target:     target,
		dispatcher: dispatcher,
	}
}

func (r *RPCServer) AddEndpoint(endpoint interface{}) error {
	return r.dispatcher.AddEndpoint(endpoint)
}

func (r *RPCServer) Start() error {
	l, err := r.transport.Listen(r.target)
	if err != nil {
		return err
	}
	r.listener = l

	ctx, stopFunc := context.WithCancel(context.Background())
	r.stopListener = stopFunc
	go r.listener.Start(ctx, r.ProcessIncoming)
	return err
}

func (r *RPCServer) Stop() error {
	r.stopListener()
	return nil
}

func (r *RPCServer) ProcessIncoming(message interfaces.Message) {
	err := message.Ack()
	if err != nil {
		log.Debugf(nil, "ack message failure, error: %s", err.Error())
		return
	}

	res, err := r.dispatcher.Dispatch(message)
	if err != nil {
		log.Debugf(nil, "handle message failure, error: %s", err.Error())
	}

	message.SetConnPool(r.transport.ConnPool)
	replyErr := message.Reply(res, err)
	if replyErr != nil {
		log.Debugf(nil, "reply message failure, error: %s", replyErr.Error())
	}
}

type RPCClient struct {
	transport  *Transport
	target     message.Target
	retry      int
	serializer interfaces.Serializer
}

func NewRPCClient(transport *Transport, target message.Target, retry int) RPCClient {
	return RPCClient{
		transport:  transport,
		target:     target,
		retry:      retry,
		serializer: &JsonSerializer{},
	}
}

func (r *RPCClient) Prepare() CallContext {
	return CallContext{
		transport:  r.transport,
		target:     r.target.Copy(),
		retry:      r.retry,
		serializer: r.serializer,
	}
}

func (r *RPCClient) PrepareAdvance(topic, host string, fanout bool) CallContext {
	target := r.target.Copy()
	if topic != "" {
		target.Topic = topic
	}
	if host != "" {
		target.Host = host
	}
	if fanout {
		target.Fanout = fanout
	}
	return CallContext{
		transport:  r.transport,
		target:     target,
		retry:      r.retry,
		serializer: r.serializer,
	}
}

type CallContext struct {
	target     message.Target
	transport  *Transport
	retry      int
	serializer interfaces.Serializer
}

func (c *CallContext) Call(ctx *contextx.Context, method string, arguments interface{}, timeout time.Duration, result interface{}) (err error) {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			switch panicErr.(type) {
			case string:
				err = errors.New(panicErr.(string))
			case error:
				err = panicErr.(error)
			}
		}
	}()

	argv, err := c.serializer.Serialize(arguments)
	if err != nil {
		return err
	}

	msg, err := c.makeMessage(ctx, method, argv)
	if err != nil {
		return err
	}
	replyMsg, err := c.transport.Send(ctx, c.target, msg, true, timeout, c.retry)
	if err != nil {
		return err
	}

	returned, err := replyMsg.GetReply()
	if err != nil {
		return err
	}

	resultValue := reflect.ValueOf(result)
	resultValueType := resultValue.Type()
	if resultValue.Kind() == reflect.Ptr {
		resultValue = resultValue.Elem()
		resultValueType = resultValue.Type()
	}

	_result, err := c.serializer.Deserialize(returned, resultValueType)
	resultValue.Set(reflect.ValueOf(_result))
	return err
}

func (c *CallContext) Cast(ctx *contextx.Context, method string, arguments interface{}) error {
	argv, err := c.serializer.Serialize(arguments)
	if err != nil {
		return err
	}

	msg, err := c.makeMessage(ctx, method, argv)
	if err != nil {
		return err
	}
	_, err = c.transport.Send(ctx, c.target, msg, false, 0, c.retry)
	return err
}

func (c *CallContext) makeMessage(ctx *contextx.Context, method string, arguments interface{}) (map[string]interface{}, error) {
	msg := map[string]interface{}{
		"method": method,
		"args":   arguments,
	}
	return msg, nil
}
