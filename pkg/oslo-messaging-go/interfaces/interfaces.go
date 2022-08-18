package interfaces

import (
	"context"
	"reflect"
	"seraph/pkg/contextx"
	"seraph/pkg/oslo-messaging-go/message"
	"time"
)

type Listener interface {
	Start(ctx context.Context, callback func(message Message))
	Stop()
}

type Dispatcher interface {
	Dispatch(message Message) (interface{}, error)
	AddEndpoint(endpoint interface{}) error
}

type Message interface {
	Ack() error
	Requeue()
	GetBody() (*message.MessageBody, error)
	Reply(reply interface{}, err error) error
	SetConnPool(pool ConnPool)
}

type Transport interface {
	SetConnPool(pool ConnPool)
	Listen(target message.Target) (Listener, error)
	ListenNotification(targets []message.Target) (Listener, error)
	Send(ctx *contextx.Context, target message.Target, msg map[string]interface{}, waitForReply bool, timeout time.Duration, retry int) (*message.MessageBody, error)
	SendNotification(ctx *contextx.Context, target message.Target, msg map[string]interface{}, retry int) error
}

type Conn interface {
	Release()
	Reset()
	Close() error
	DeclareTopicConsumer(exchange, topic string, callback func(message Message), queue string)
	DeclareFanoutConsumer(topic string, callback func(message Message))
	DeclareDirectConsumer(topic string, callback func(message Message))

	DirectSend(msgId string, msg message.MessageBody, retry int) error
	TopicSend(exchange, topic string, msg message.MessageBody, timeout time.Duration, retry int) error
	FanoutSend(topic string, msg message.MessageBody, retry int) error
	NotifySend(exchange, topic string, msg message.MessageBody, retry int) error

	Consume(timeout time.Duration)
}

type ConnPool interface {
	Initialize() error

	SetPoolTimeout(timeout time.Duration)
	SetMaxCount(count int)
	SetMaxIdleCount(count int)
	SetMinIdleCount(count int)
	SetIdleTimeout(timeout time.Duration)

	GetConnection(ctx context.Context, purpose string) (Conn, error)

	WithConnection(ctx context.Context, purpose string, callback func(conn Conn) (interface{}, error)) (interface{}, error)
}

type Dialector interface {
	Name() string
	Initialize(transport Transport) error
}

type Logger interface {
	Info(args ...interface{})
	Debug(args ...interface{})
	Trace(args ...interface{})
	Warn(args ...interface{})
	Panic(args ...interface{})
	Error(args ...interface{})

	Infof(format string, args ...interface{})
	Debugf(format string, args ...interface{})
	Tracef(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Panicf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

type Serializer interface {
	Serialize(data interface{}) (interface{}, error)
	Deserialize(data interface{}, argType reflect.Type) (interface{}, error)
}
