package messaging

import (
	"context"
	"seraph/pkg/oslo-messaging-go/interfaces"
	"sync/atomic"
	"time"
)

type RpcAMQPListener struct {
	messages         chan interfaces.Message
	conn             interfaces.Conn
	incomingCallback func(message interfaces.Message)
	cancelFunc       context.CancelFunc
	stopped          uint32
}

func (r *RpcAMQPListener) Incoming(msg interfaces.Message) {
	r.messages <- msg
}

func (r *RpcAMQPListener) ServeIncoming(ctx context.Context) {
	for {
		select {
		case msg := <-r.messages:
			r.incomingCallback(msg)
		case <-ctx.Done():
			return
		}
	}
}

func (r *RpcAMQPListener) Start(ctx context.Context, callback func(message interfaces.Message)) {
	r.incomingCallback = callback
	serveCtx, cancel := context.WithCancel(ctx)
	r.cancelFunc = cancel
	go r.Poll(ctx)
	r.ServeIncoming(serveCtx)
}

func (r *RpcAMQPListener) Poll(ctx context.Context) {
	for {
		r.conn.Consume(5 * time.Second)
		if r.IsStop() {
			return
		}
	}
}

func (r *RpcAMQPListener) Stop() {
	r.cancelFunc()
	_ = r.conn.Close()
	atomic.StoreUint32(&r.stopped, 1)
}

func (r *RpcAMQPListener) IsStop() bool {
	return atomic.LoadUint32(&r.stopped) == 1
}

func NewRpcAMQPListener(conn interfaces.Conn) RpcAMQPListener {
	return RpcAMQPListener{
		conn:     conn,
		messages: make(chan interfaces.Message, 0),
	}
}
