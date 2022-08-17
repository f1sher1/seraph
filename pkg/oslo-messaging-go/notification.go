package messaging

import (
	"context"
	"seraph/pkg/log"
	"seraph/pkg/oslo-messaging-go/interfaces"
	"seraph/pkg/oslo-messaging-go/message"
)

const (
	requeueNotificationFlag = "REQUEUE"
)

type NotificationServer struct {
	transport    *Transport
	targets      []message.Target
	dispatcher   interfaces.Dispatcher
	listener     interfaces.Listener
	stopListener context.CancelFunc
}

func NewNotificationServer(transport *Transport, targets []message.Target, dispatcher interfaces.Dispatcher) NotificationServer {
	return NotificationServer{
		transport:  transport,
		targets:    targets,
		dispatcher: dispatcher,
	}
}

func (r *NotificationServer) Start() error {
	l, err := r.transport.ListenNotification(r.targets)
	if err != nil {
		return err
	}
	r.listener = l

	ctx, stopFunc := context.WithCancel(context.Background())
	r.stopListener = stopFunc
	r.listener.Start(ctx, r.ProcessIncoming)
	return err
}

func (r *NotificationServer) Stop() error {
	r.stopListener()
	return nil
}

func (r *NotificationServer) ProcessIncoming(message interfaces.Message) {
	res, err := r.dispatcher.Dispatch(message)
	if err != nil {
		log.Debugf(nil, "handle message failure, error: %s", err.Error())
		message.Requeue()
		return
	}

	switch res.(type) {
	case string:
		if res.(string) == requeueNotificationFlag {
			message.Requeue()
			return
		}
	}
	err = message.Ack()
	if err != nil {
		log.Debugf(nil, "ack message failure, error: %s", err.Error())
		return
	}
}
