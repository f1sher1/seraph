package rabbit

import (
	"fmt"
	"seraph/pkg/oslo-messaging-go/message"
	"strconv"
	"time"

	"github.com/streadway/amqp"
)

type Publisher struct {
	channel *AMQPChannel
	conn    *Connection
}

func (p Publisher) prepareMessage(body []byte, contentType, contentEncoding string, priority uint8, properties, headers amqp.Table, expiration time.Duration) amqp.Publishing {
	msg := amqp.Publishing{
		Headers:         headers,
		ContentType:     contentType,
		ContentEncoding: contentEncoding,
		DeliveryMode:    0,
		Priority:        priority,
		Body:            body,
	}
	if properties != nil {
		if replyTo, ok := properties["replyTo"]; ok {
			msg.ReplyTo = replyTo.(Queue).name
		}
	}

	if expiration > 0 {
		msg.Expiration = strconv.Itoa(int(expiration / time.Millisecond))
	}
	return msg
}

func (p Publisher) publish(body []byte, routingKey, contentType, contentEncoding string, priority uint8, exchange Exchange, timeout time.Duration, properties, headers amqp.Table, retry int) error {
	msg := p.prepareMessage(body, contentType, contentEncoding, priority, properties, headers, timeout)

	if retry > 0 {
		err := p.conn.EnsureWithRetry(func() *message.Error {
			err := p.channel.Publish(exchange.name, routingKey, false, false, msg)
			if err != nil {
				return message.NewError(0, err.Error(), false, false)
			} else {
				return nil
			}
		}, retry, nil, nil)
		if err == nil {
			return err
		}
		return fmt.Errorf("publish failed, error: %s", err.Error())
	} else {
		return p.channel.Publish(exchange.name, routingKey, false, false, msg)
	}

}

func (p Publisher) Publish(msg message.MessageBody, exchange Exchange, routingKey string, timeout time.Duration, retry int) error {
	bytes, err := msg.ToBytes()
	if err != nil {
		return err
	}
	return p.publish(bytes, routingKey, "application/json", "utf-8", 0, exchange, timeout, nil, nil, retry)
}

func NewPublisher(conn *Connection, channel *AMQPChannel) *Publisher {
	return &Publisher{
		channel: channel,
		conn:    conn,
	}
}
