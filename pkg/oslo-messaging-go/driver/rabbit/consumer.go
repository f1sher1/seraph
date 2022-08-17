package rabbit

import (
	"context"
	"seraph/pkg/log"
	"seraph/pkg/oslo-messaging-go/interfaces"
	"seraph/pkg/oslo-messaging-go/message"
	"sync/atomic"

	"github.com/streadway/amqp"
)

type Exchange struct {
	name       string
	kind       string
	durable    bool
	autoDelete bool
	passive    bool
}

func (e *Exchange) Declare(nowait bool, channel *AMQPChannel) error {
	if e.name == "" {
		return nil
	}
	if e.passive {
		return channel.ExchangeDeclarePassive(e.name, e.kind, e.durable, e.autoDelete, false, nowait, nil)
	}
	return channel.ExchangeDeclare(e.name, e.kind, e.durable, e.autoDelete, false, nowait, nil)
}

type Queue struct {
	name       string
	channel    *AMQPChannel
	exchange   *Exchange
	durable    bool
	autoDelete bool
	routingKey string
	arguments  amqp.Table

	notDeclared bool
	consuming   uint32

	stopFunc context.CancelFunc
}

func (q *Queue) Declare(channel *AMQPChannel, nowait bool) error {
	q.channel = channel
	if err := q.DeclareExchange(nowait); err != nil {
		return err
	}
	if err := q.DeclareQueue(nowait); err != nil {
		return err
	}
	return nil
}

func (q *Queue) DeclareExchange(nowait bool) error {
	return q.exchange.Declare(nowait, q.channel)
}

func (q *Queue) DeclareQueue(nowait bool) error {
	_, err := q.channel.QueueDeclare(q.name, q.durable, q.autoDelete, false, nowait, q.arguments)
	if err != nil {
		return err
	}
	if q.exchange != nil && q.exchange.name != "" {
		return q.QueueBind(nowait)
	}
	return nil
}

func (q *Queue) QueueBind(nowait bool) error {
	return q.channel.QueueBind(q.name, q.routingKey, q.exchange.name, nowait, nil)
}

func (q *Queue) Cancel(id string, nowait bool) error {
	q.stopFunc()
	return q.channel.Cancel(id, nowait)
}

func (q *Queue) Delete(ifUnused bool, ifEmpty bool, nowait bool) error {
	_, err := q.channel.QueueDelete(q.name, ifUnused, ifEmpty, nowait)
	return err
}

func (q *Queue) Consume(id string, callback func(message interfaces.Message), nowait bool) error {
	deliveryChan, err := q.channel.Consume(q.name, id, false, false, false, nowait, nil)
	if err != nil {
		log.Debugf(nil, "consume channel %s error: %s", id, err.Error())
		return err
	}

	ctx, stopFunc := context.WithCancel(context.Background())
	go q.ConsumeMessages(ctx, deliveryChan, callback)
	q.stopFunc = stopFunc
	return nil
}

func (q *Queue) ConsumeMessages(ctx context.Context, deliveryChan <-chan amqp.Delivery, callback func(message interfaces.Message)) {
	for {
		select {
		case msg := <-deliveryChan:
			if msg.Body == nil {
				goto END
			}
			go q.execCallback(callback, msg)
		case <-ctx.Done():
			goto END
		}
	}
END:
	//for msg := range deliveryChan {
	//	go q.execCallback(callback, msg)
	//}
	log.Debugf(nil, "consume message ended on %s, maybe channel closed or canceled", q.name)
}

func (q Queue) execCallback(callback func(message interfaces.Message), message amqp.Delivery) {
	rabbitMsg, err := NewMessage(message)
	if err != nil {
		log.Debugf(nil, "parse message failed, error: %s", err.Error())
		err = message.Reject(true)
		if err != nil {
			log.Errorf(nil, "requeue message failed, error: %s, message: %s", err.Error(), message.Body)
		}
	}
	// log.Debugf("!1 %+v & %#v | %#v", callback, runtime.FuncForPC(reflect.ValueOf(callback).Pointer()).Name(), rabbitMsg.Payload.Body)
	callback(rabbitMsg)
}

func (q *Queue) IsConsuming() bool {
	return atomic.LoadUint32(&q.consuming) == 1
}

func (q *Queue) StartConsuming() {
	atomic.StoreUint32(&q.consuming, 1)
}

func (q *Queue) StopConsuming() {
	atomic.StoreUint32(&q.consuming, 0)
}

type Consumer struct {
	ID string

	exchangeName       string
	exchangeAutoDelete bool

	queueName       string
	queueAutoDelete bool
	queueHAEnable   bool
	queueTTL        int

	routingKey string
	kind       string
	durable    bool

	callback func(message interfaces.Message)

	queue      *Queue
	exchange   *Exchange
	declaredOn *AMQPChannel
}

func NewConsumer(kind, exchangeName, queueName, routingKey string, durable, exchangeAutoDelete, queueAutoDelete, queueHAEnable bool, queueTTL int, callback func(message interfaces.Message)) *Consumer {
	return &Consumer{
		exchangeName:       exchangeName,
		exchangeAutoDelete: exchangeAutoDelete,
		queueName:          queueName,
		queueAutoDelete:    queueAutoDelete,
		queueHAEnable:      queueHAEnable,
		queueTTL:           queueTTL,

		routingKey: routingKey,
		kind:       kind,
		durable:    durable,

		callback: callback,

		exchange: &Exchange{
			name:       exchangeName,
			kind:       kind,
			durable:    durable,
			autoDelete: exchangeAutoDelete,
		},
	}
}

func getQueueArguments(queueHAEnable bool, queueTTL int) amqp.Table {
	arguments := amqp.Table{}
	if queueHAEnable {
		arguments["x-ha-policy"] = "all"
	}
	if queueTTL > 0 {
		arguments["x-expires"] = queueTTL * 1000
	}
	return arguments
}

func (consumer *Consumer) Declare(channel *AMQPChannel) error {
	log.Debugf(nil, "declaring consumer %s on connection %s", consumer.queueName, channel.Conn.ID)
	//channel := conn.GetChannel()
	//if channel == nil {
	//	errStr := fmt.Sprintf("declare consumer %s on a closed connection %s", consumer.queueName, conn.ID)
	//	log.Error(errStr)
	//	return errors.New(errStr)
	//}

	consumer.queue = &Queue{
		name:       consumer.queueName,
		exchange:   consumer.exchange,
		durable:    consumer.durable,
		autoDelete: consumer.queueAutoDelete,
		routingKey: consumer.routingKey,
		arguments:  getQueueArguments(consumer.queueHAEnable, consumer.queueTTL),
	}
	log.Debugf(nil, "connection: %s, queue.declare: %s", channel.ID, consumer.queueName)
	err := consumer.queue.Declare(channel, false)
	if err != nil {
		log.Errorf(nil, "connection: %s, queue.declare: %s failed, error: %s", channel.Conn.ID, consumer.queueName, err.Error())
		return err
	}

	consumer.declaredOn = channel
	return nil
}

func (consumer *Consumer) Cancel() {
	if consumer.queue != nil {
		log.Debugf(nil, "cancel consumer %s", consumer.ID)
		err := consumer.queue.Cancel(consumer.ID, false)
		if err != nil {
			log.Debugf(nil, "cancel consumer %s failed, error: %s", consumer.ID, err.Error())
		}
	}
}

func (consumer *Consumer) Consume(channel *AMQPChannel) *message.Error {
	log.Debugf(nil, "consuming consumer %s", consumer.ID)
	//channel, err := conn.CreateChannel()
	//
	//if err != nil {
	//	errStr := fmt.Sprintf("consume consumer %s on a closed connection %s", consumer.queueName, conn.ID)
	//	log.Error(errStr)
	//	return NewError(amqp.InternalError, errStr, false)
	//}

	if consumer.declaredOn == nil || consumer.declaredOn.ID != channel.ID {
		log.Debugf(nil, "consumer %s[queue=%s] declared channel changed", consumer.ID, consumer.queueName)
		if consumer.declaredOn != nil {
			_ = consumer.declaredOn.SafeClose()
		}
		if err := consumer.Declare(channel); err != nil {
			return parseError(err)
		}
	}

	err := consumer.queue.Consume(consumer.ID, consumer.callback, false)
	if err != nil {
		parsedErr := parseError(err)

		if parsedErr.Code == 404 {
			log.Debugf(nil, "consume queue on %s failed, maybe queue canceled on MQ, error: %s", consumer.ID, parsedErr.Error())
			if err := consumer.Declare(channel); err != nil {
				log.Debugf(nil, "redeclare queue on %s failed, error: %s", consumer.ID, err.Error())
				return NewError(amqp.InternalError, err.Error(), false)
			}

			err = consumer.queue.Consume(consumer.ID, consumer.callback, false)
			parsedErr = parseError(err)
			if parsedErr != nil {
				log.Debugf(nil, "reconsume queue on %s failed, maybe connection was broken, error: %s", consumer.ID, parsedErr.Error())
				return parsedErr
			}
			return nil
		} else {
			log.Debugf(nil, "consume queue on %s failed, maybe connection was broken, error: %s", consumer.ID, parsedErr.Error())
			channel.Conn.resetConsumed()
		}
		return parsedErr
	}

	return nil
}
