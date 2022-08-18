package rabbit

import (
	"context"
	"seraph/pkg/oslo-messaging-go/interfaces"
	"seraph/pkg/oslo-messaging-go/message"

	"github.com/streadway/amqp"
)

type Message struct {
	Payload    *message.MessageBody
	RawMessage amqp.Delivery
	connPool   interfaces.ConnPool
}

func (m *Message) SetConnPool(pool interfaces.ConnPool) {
	m.connPool = pool
}

func (m *Message) Reply(reply interface{}, replyErr error) error {
	// todo;
	payload, err := m.Payload.BuildReply(reply, replyErr, true)
	if err != nil {
		return err
	}

	if payload.ReplyQueue != "" {
		_, err = m.connPool.WithConnection(context.Background(), message.PurposeSend, func(conn interfaces.Conn) (interface{}, error) {
			return nil, conn.DirectSend(payload.ReplyQueue, payload, 0)
		})

		return err
	}
	return nil
}

func (m *Message) Ack() error {
	return m.RawMessage.Ack(false)
}

func (m *Message) Requeue() {
	_ = m.RawMessage.Reject(true)
}

func (m *Message) Initialize() error {
	payload, err := message.ParseMessageBody(m.RawMessage.Body)
	if err != nil {
		return err
	}
	m.Payload = payload
	return nil
}

func (m *Message) GetBody() (*message.MessageBody, error) {
	return m.Payload, nil
}

func NewMessage(rawMessage amqp.Delivery) (*Message, error) {
	msg := &Message{RawMessage: rawMessage}
	err := msg.Initialize()
	if err != nil {
		return nil, err
	}
	return msg, nil
}
