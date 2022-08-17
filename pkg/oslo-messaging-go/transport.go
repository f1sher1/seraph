package messaging

import (
	"context"
	"errors"
	"fmt"
	"seraph/pkg/contextx"
	"seraph/pkg/log"
	"seraph/pkg/oslo-messaging-go/interfaces"
	"seraph/pkg/oslo-messaging-go/message"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

type ReplyWaiter struct {
	conn       interfaces.Conn
	replyQueue string
	messages   chan interface{}
	finished   uint32
}

func (r *ReplyWaiter) Initialize() error {
	r.conn.DeclareDirectConsumer(r.replyQueue, r.Incoming)
	return nil
}

func (r *ReplyWaiter) Incoming(msg interfaces.Message) {
	_ = msg.Ack()
	body, err := msg.GetBody()
	if err != nil {
		r.messages <- err
	} else {
		r.messages <- body
	}
}

func (r *ReplyWaiter) Finished() {
	log.Debugf(nil, "reply finished")
	atomic.StoreUint32(&r.finished, 1)
	_ = r.conn.Close()
}

func (r *ReplyWaiter) IsFinished() bool {
	return atomic.LoadUint32(&r.finished) == 1
}

func (r *ReplyWaiter) Poll() {
	for {
		if !r.IsFinished() {
			r.conn.Consume(5 * time.Second)
		} else {
			return
		}
	}
}

func (r *ReplyWaiter) Wait(timeout time.Duration) (b *message.MessageBody, err error) {
	if timeout > 0 {
		select {
		case msg := <-r.messages:
			if body, ok := msg.(*message.MessageBody); ok {
				b = body
			} else {
				err = msg.(error)
			}
			goto END
		case <-time.Tick(timeout):
			err = fmt.Errorf("wait %s timeout", r.replyQueue)
			goto END
		}
	} else {
		select {
		case msg := <-r.messages:
			if body, ok := msg.(*message.MessageBody); ok {
				b = body
			} else {
				err = msg.(error)
			}
			goto END
		}
	}
END:
	r.Finished()
	return b, err
}

func NewReplyWaiter(transport *Transport, replyQueue string) (*ReplyWaiter, error) {
	conn, err := transport.ConnPool.GetConnection(context.Background(), message.PurposeListen)
	if err != nil {
		return nil, err
	}

	waiter := &ReplyWaiter{
		conn:       conn,
		replyQueue: replyQueue,
		messages:   make(chan interface{}),
	}
	err = waiter.Initialize()
	if err != nil {
		return nil, err
	}

	go waiter.Poll()
	return waiter, nil
}

type Transport struct {
	replyMu  sync.Mutex
	ConnPool interfaces.ConnPool
}

func (t *Transport) Listen(target message.Target) (interfaces.Listener, error) {
	conn, err := t.ConnPool.GetConnection(context.Background(), message.PurposeListen)
	if err != nil {
		return nil, err
	}

	listener := NewRpcAMQPListener(conn)
	conn.DeclareTopicConsumer(target.Exchange, target.Topic, listener.Incoming, "")

	if target.Host != "" {
		conn.DeclareTopicConsumer(target.Exchange, fmt.Sprintf("%s.%s", target.Topic, target.Host), listener.Incoming, "")
	}

	conn.DeclareFanoutConsumer(target.Topic, listener.Incoming)

	return &listener, nil
}

func (t *Transport) ListenNotification(targets []message.Target) (interfaces.Listener, error) {
	conn, err := t.ConnPool.GetConnection(context.Background(), message.PurposeListen)
	if err != nil {
		return nil, err
	}

	listener := NewRpcAMQPListener(conn)

	for _, target := range targets {
		topic := fmt.Sprintf("%s.%s", target.Topic, target.Priority)
		conn.DeclareTopicConsumer(target.Exchange, topic, listener.Incoming, "")
	}

	return &listener, nil
}

func (t *Transport) Send(ctx *contextx.Context, target message.Target, msg map[string]interface{}, waitForReply bool, timeout time.Duration, retry int) (*message.MessageBody, error) {
	if target.Topic == "" {
		return nil, errors.New("A topic is required to send")
	}

	var waiter *ReplyWaiter
	var err error

	rpcMsg := message.NewMessageBody(ctx, msg, target.Version)
	if waitForReply {
		rpcMsg.MessageId = uuid.NewString()
		rpcMsg.ReplyQueue = fmt.Sprintf("reply_%s", strings.ReplaceAll(uuid.NewString(), "-", ""))
		waiter, err = NewReplyWaiter(t, rpcMsg.ReplyQueue)
		if err != nil {
			return nil, err
		}
	}

	result, err := t.ConnPool.WithConnection(context.Background(), message.PurposeSend, func(conn interfaces.Conn) (interface{}, error) {
		if err != nil {
			if waiter != nil {
				waiter.Finished()
			}
			return nil, err
		}

		if target.Notify {
			log.Debugf(nil, "Notify exchange %s topic %s", target.Exchange, target.Topic)
			err = conn.NotifySend(target.Exchange, target.Topic, rpcMsg, retry)
			if err != nil {
				log.Debugf(nil, "Notify exchange %s topic %s failure, error: %s", target.Exchange, target.Topic, err.Error())
			}
		} else if target.Fanout {
			log.Debugf(nil, "Fanout topic %s", target.Topic)
			err = conn.FanoutSend(target.Topic, rpcMsg, retry)
			if err != nil {
				log.Debugf(nil, "Fanout topic %s failure, error: %s", target.Topic, err.Error())
			}
		} else {
			topic := target.Topic
			if target.Host != "" {
				topic = fmt.Sprintf("%s.%s", topic, target.Host)
			}
			log.Debugf(nil, "exchange %s topic %s", target.Exchange, target.Topic)
			err = conn.TopicSend(target.Exchange, target.Topic, rpcMsg, timeout, retry)
			if err != nil {
				log.Debugf(nil, "exchange %s topic %s failure, error: %s", target.Exchange, target.Topic, err.Error())
			}
		}
		if err != nil {
			if waiter != nil {
				waiter.Finished()
			}
			return nil, err
		} else {
			if waiter != nil {
				return waiter.Wait(timeout)
			}
			return nil, nil
		}
	})

	if err != nil {
		return nil, err
	} else {
		return result.(*message.MessageBody), nil
	}
}

func (t *Transport) SendNotification(ctx *contextx.Context, target message.Target, msg map[string]interface{}, retry int) error {
	if target.Topic == "" {
		return errors.New("A topic is required to send")
	}

	target.Notify = true
	_, err := t.Send(ctx, target, msg, false, 0, retry)
	return err
}

func (t *Transport) SetConnPool(pool interfaces.ConnPool) {
	t.ConnPool = pool
}

func NewTransport(dialector interfaces.Dialector) (*Transport, error) {
	transport := &Transport{}
	err := dialector.Initialize(transport)
	if err != nil {
		return nil, err
	}
	return transport, nil
}
