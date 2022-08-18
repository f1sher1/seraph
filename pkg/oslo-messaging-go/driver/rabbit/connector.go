package rabbit

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"seraph/pkg/log"
	"seraph/pkg/oslo-messaging-go/interfaces"
	"seraph/pkg/oslo-messaging-go/message"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/streadway/amqp"
)

const (
	defaultHeartbeat = 10 * time.Second
	defaultLocale    = "en_US"
	defaultProduct   = "rabbitmqx"
	defaultVersion   = "0.1"
)

type ConnectionLock struct {
	mu      sync.Mutex
	purpose string
}

func (c *ConnectionLock) Lock() {
	if c.purpose == message.PurposeSend {
		c.mu.Lock()
	}
}

func (c *ConnectionLock) Unlock() {
	if c.purpose == message.PurposeSend {
		c.mu.Unlock()
	}
}

type AMQPChannel struct {
	ID   string
	Conn *Connection
	*amqp.Channel
	Closed        bool
	cancelChannel chan string
	eventChannel  chan *amqp.Error

	connEventCancel chan *amqp.Error
	stopWatch       context.CancelFunc
}

func (c *AMQPChannel) WatchState() {
	c.cancelChannel = make(chan string)
	c.eventChannel = make(chan *amqp.Error)
	c.Channel.NotifyClose(c.eventChannel)
	c.Channel.NotifyCancel(c.cancelChannel)

	watchCtx, stopWatch := context.WithCancel(context.Background())
	c.stopWatch = stopWatch

	go c.watchStateLoop(watchCtx)
}

func (c *AMQPChannel) watchStateLoop(ctx context.Context) {
	for {
		select {
		case e := <-c.eventChannel:
			c.Closed = true
			if e == nil {
				return
			}
			log.Warnf(nil, "connection %s got event %s, recoverable %v 1", c.Conn.ID, e.Error(), e.Recover)
		case queueName := <-c.cancelChannel:
			log.Warnf(nil, "connection %s got queue cancel event, queue: %s", c.Conn.ID, queueName)
			c.Closed = true
			c.connEventCancel <- &amqp.Error{
				Code:    amqp.NotFound,
				Recover: true,
				Server:  true,
				Reason:  queueName,
			}
		case <-ctx.Done():
			return
		}
	}
}

func (c *AMQPChannel) SetConn(conn *Connection) {
	c.Conn = conn
}

func (c *AMQPChannel) SafeClose() (err error) {
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
	c.stopWatch()
	return c.Channel.Close()
}

type AMQPConnection struct {
	ID           string
	amqpConn     *amqp.Connection
	eventChannel chan *amqp.Error
	blockChannel chan amqp.Blocking
	stopWatch    context.CancelFunc

	dialInterval time.Duration
	url          string
	heartbeat    time.Duration
	locale       string
	properties   amqp.Table

	connected uint32
	closed    uint32

	channel *AMQPChannel
	conn    *Connection
}

func (c *AMQPConnection) SetConnected() {
	atomic.StoreUint32(&c.connected, 1)
}

func (c *AMQPConnection) SetDisconnected() {
	atomic.StoreUint32(&c.connected, 0)
}

func (c *AMQPConnection) IsConnected() bool {
	return atomic.LoadUint32(&c.connected) == 1
}

func (c *AMQPConnection) IsClosed() bool {
	return atomic.LoadUint32(&c.closed) == 1
}

func (c *AMQPConnection) Close() (err error) {
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

	log.Debugf(nil, "close connection %s", c.ID)
	if c.IsClosed() {
		return nil
	}
	atomic.StoreUint32(&c.closed, 1)

	c.stopWatch()

	if c.amqpConn != nil && !c.amqpConn.IsClosed() {
		return c.amqpConn.Close()
	}
	return nil
}

func (c *AMQPConnection) RecreateChannel() error {
	c.channel = nil
	channel, err := c.CreateChannel()
	if err != nil {
		log.Debugf(nil, "create channel failed on connection %s, error: %s", c.ID, err.Error())
		c.SetDisconnected()
		return err
	}
	c.channel = channel
	return nil
}

func (c *AMQPConnection) GetChannel() *AMQPChannel {
	return c.channel
}

func (c *AMQPConnection) CreateChannel() (*AMQPChannel, error) {
	channel, err := c.amqpConn.Channel()
	if err != nil {
		return nil, err
	}
	ch := &AMQPChannel{
		ID:              uuid.NewString(),
		Channel:         channel,
		Conn:            c.conn,
		connEventCancel: c.eventChannel,
	}
	ch.WatchState()
	return ch, nil
}

func (c *AMQPConnection) EnsureConnection() error {
	if c.IsConnected() {
		return nil
	}

	log.Debugf(nil, "connection %s connecting", c.ID)

	dialCfg := amqp.Config{
		Heartbeat: c.heartbeat,
		Locale:    c.locale,
		Properties: amqp.Table{
			"product": c.properties["product"],
			"version": c.properties["version"],
		},
	}

	for {
		if c.IsClosed() {
			return errors.New("closed")
		}

		amqpConn, err := amqp.DialConfig(c.url, dialCfg)
		if err != nil {
			log.Errorf(nil, "connection %s connect failed, error: %s", c.ID, err.Error())
			time.Sleep(c.dialInterval)
			continue
		}

		c.amqpConn = amqpConn
		c.eventChannel = make(chan *amqp.Error)
		c.blockChannel = make(chan amqp.Blocking)

		err = c.RecreateChannel()
		if err != nil {
			log.Errorf(nil, "connection %s create channel failed, error: %s", c.ID, err.Error())
			_ = c.amqpConn.Close()
			time.Sleep(c.dialInterval)
			continue
		}

		amqpConn.NotifyClose(c.eventChannel)
		amqpConn.NotifyBlocked(c.blockChannel)

		watchCtx, stopWatch := context.WithCancel(context.Background())
		go c.WatchState(watchCtx)

		c.stopWatch = stopWatch
		break
	}

	c.SetConnected()
	log.Debugf(nil, "connection %s connected", c.ID)
	return nil
}

func (c *AMQPConnection) WatchState(ctx context.Context) {
	for {
		select {
		case b := <-c.blockChannel:
			if b.Active {
				log.Infof(nil, "connection %s is activate", c.ID)
			} else {
				log.Warnf(nil, "connection %s was blocked, reason: %s", c.ID, b.Reason)
				time.Sleep(5 * time.Second)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (c *AMQPConnection) DrainEvent(timeout time.Duration) *message.Error {
	select {
	case e := <-c.eventChannel:
		if e == nil {
			return nil
		}
		log.Warnf(nil, "connection %s got event %s, recoverable %v", c.ID, e.Error(), e.Recover)
		if e.Code == amqp.NotFound {
			queueName := e.Reason
			c.conn.resetConsumedByName(queueName)
			return nil
		} else {
			c.SetDisconnected()
		}
		return message.NewError(e.Code, e.Reason, e.Recover, e.Server)
	case <-time.Tick(timeout):
		return nil
	}
}

type Connection struct {
	ID       string
	pooled   bool
	pool     *ConnPool
	amqpConn *AMQPConnection

	createdAt time.Time
	usedAt    time.Time

	purpose    string
	url        string
	heartbeat  time.Duration
	locale     string
	properties amqp.Table

	connectionMu ConnectionLock
	consumers    map[*Consumer]*AMQPChannel

	closed uint32

	channel  *AMQPChannel
	producer *Publisher

	consumed map[string]bool
}

func (c *Connection) resetConsumed() {
	c.consumed = map[string]bool{}
}

func (c *Connection) resetConsumedByName(name string) {
	if _, ok := c.consumed[name]; ok {
		delete(c.consumed, name)
	}
}

func (c *Connection) consume(timeout time.Duration) *message.Error {
	c.connectionMu.Lock()
	defer c.connectionMu.Unlock()

	if c.IsClosed() {
		return nil
	}

	for consumer, ch := range c.consumers {
		if _, ok := c.consumed[consumer.ID]; !ok {
			if ch.Closed {
				_ch, err := c.CreateChannel()
				if err != nil {
					return &message.Error{
						Code:    amqp.ChannelError,
						Reason:  err.Error(),
						Server:  true,
						Recover: false,
					}
				}
				ch = _ch
			}
			err := consumer.Consume(ch)
			if err != nil {
				return err
			}
			c.consumed[consumer.ID] = true
			c.consumers[consumer] = ch
		}
	}

	return c.amqpConn.DrainEvent(timeout)
}

func (c *Connection) Consume(timeout time.Duration) {
	c.connectionMu.Lock()
	defer c.connectionMu.Unlock()

	if c.IsClosed() {
		return
	}

	_ = c.Ensure(func() *message.Error {
		return c.consume(timeout)
	}, func(m *message.Error) {
		c.resetConsumed()
	}, nil)
}

func (c *Connection) Publish(exchange Exchange, msg message.MessageBody, routingKey string, timeout time.Duration, retry int) error {
	exchangeName := exchange.name
	if exchangeName == "" {
		exchangeName = "default"
	}

	if !exchange.passive {
		channel := c.GetChannel()
		if channel == nil {
			log.Errorf(nil, "the publisher using a disconnected connection %s", c.ID)
			return errors.New("the publisher using a disconnected connection")
		}
		err := exchange.Declare(false, channel)
		if err != nil {
			log.Errorf(nil, "the publisher declare exchange '%s' on connection %s failed, error: %s", exchangeName, c.ID, err.Error())
			return err
		}
	}

	log.Debugf(nil, "connection.publish: sending message to exchange '%s' with routing key '%s'", exchangeName, routingKey)
	return c.producer.Publish(msg, exchange, routingKey, timeout, retry)
}

func (c *Connection) PublishAndCreateQueue(exchange Exchange, msg message.MessageBody, routingKey string, timeout time.Duration, retry int) error {
	channel := c.GetChannel()
	if channel == nil {
		log.Errorf(nil, "the notifier using a disconnected connection %s", c.ID)
		return errors.New("the notifier using a disconnected connection")
	}

	queueHa := false
	if queueHaV, ok := c.properties["amqp_ha_queues"]; ok {
		queueHa = queueHaV.(bool)
	}

	queue := Queue{
		name:        routingKey,
		exchange:    &exchange,
		durable:     exchange.durable,
		autoDelete:  exchange.autoDelete,
		routingKey:  routingKey,
		arguments:   getQueueArguments(queueHa, 0),
		notDeclared: false,
	}
	err := queue.Declare(channel, false)
	if err != nil {
		return err
	}

	return c.Publish(exchange, msg, routingKey, timeout, retry)
}

func (c *Connection) DirectSend(msgId string, msg message.MessageBody, retry int) error {
	// default exchange
	exchange := Exchange{
		name:       "",
		kind:       amqp.ExchangeDirect,
		durable:    false,
		autoDelete: true,
		passive:    true,
	}

	return c.Publish(exchange, msg, msgId, 0, retry)
}

func (c *Connection) TopicSend(exchangeName, topic string, msg message.MessageBody, timeout time.Duration, retry int) error {
	durable := false
	autoDelete := false
	if durableV, ok := c.properties["amqp_durable_queues"]; ok {
		durable = durableV.(bool)
	}
	if autoDeleteV, ok := c.properties["amqp_auto_delete"]; ok {
		autoDelete = autoDeleteV.(bool)
	}

	exchange := Exchange{
		name:       exchangeName,
		kind:       amqp.ExchangeTopic,
		durable:    durable,
		autoDelete: autoDelete,
	}

	return c.Publish(exchange, msg, topic, timeout, retry)
}

func (c *Connection) FanoutSend(topic string, msg message.MessageBody, retry int) error {
	exchange := Exchange{
		name:       fmt.Sprintf("%s_fanout", topic),
		kind:       amqp.ExchangeTopic,
		durable:    false,
		autoDelete: true,
	}

	return c.Publish(exchange, msg, topic, 0, retry)
}

func (c *Connection) NotifySend(exchangeName, topic string, msg message.MessageBody, retry int) error {
	durable := false
	autoDelete := false

	if durableV, ok := c.properties["amqp_durable_queues"]; ok {
		durable = durableV.(bool)
	}
	if autoDeleteV, ok := c.properties["amqp_auto_delete"]; ok {
		autoDelete = autoDeleteV.(bool)
	}

	exchange := Exchange{
		name:       exchangeName,
		kind:       amqp.ExchangeTopic,
		durable:    durable,
		autoDelete: autoDelete,
	}

	return c.PublishAndCreateQueue(exchange, msg, topic, 0, retry)
}

func (c *Connection) EnsureConnection() {
	c.connectionMu.Lock()
	defer c.connectionMu.Unlock()
	_ = c.amqpConn.EnsureConnection()
}

func (c *Connection) Initialize(ctx context.Context) error {
	c.resetConsumed()
	c.amqpConn = &AMQPConnection{
		ID:           c.ID,
		url:          c.url,
		heartbeat:    c.heartbeat,
		locale:       c.locale,
		properties:   c.properties,
		dialInterval: 3 * time.Second,
		conn:         c,
	}
	c.EnsureConnection()
	c.SetChannel(c.amqpConn.GetChannel())

	if c.purpose == message.PurposeSend {
		go c.doHeartbeat()
	}
	return nil
}

func (c *Connection) drainEvents(timeout time.Duration) *message.Error {
	return c.amqpConn.DrainEvent(timeout)
}

func (c *Connection) IsClosed() bool {
	return atomic.LoadUint32(&c.closed) == 1
}

func (c *Connection) Close() error {
	atomic.StoreUint32(&c.closed, 1)
	if c.amqpConn != nil && !c.amqpConn.IsClosed() {
		for consumer := range c.consumers {
			if consumer.kind == amqp.ExchangeFanout {
				_ = consumer.queue.Delete(false, false, false)
			}
		}
		c.SetChannel(nil)
		return c.amqpConn.Close()
	}
	return nil
}

func (c *Connection) Reset() {
	c.connectionMu.Lock()
	defer c.connectionMu.Unlock()

	for consumer := range c.consumers {
		consumer.Cancel()
	}
	c.consumers = map[*Consumer]*AMQPChannel{}
}

func (c *Connection) Release() {
	if c.pooled && c.pool != nil {
		c.Reset()
		c.pool.Put(c)
	}
	_ = c.Close()
}

func (c *Connection) SetChannel(channel *AMQPChannel) {
	if channel != nil && c.channel != nil && channel.ID == c.channel.ID {
		return
	}

	if c.channel != nil {
		_ = c.channel.SafeClose()
	}
	c.channel = channel

	if channel != nil {
		log.Debugf(nil, "channel changed on connection %s", c.ID)
		c.producer = NewPublisher(c, channel)
		for consumer := range c.consumers {
			ch, err := c.CreateChannel()
			if err != nil {
				log.Errorf(nil, "create channel failed, error: %s", err.Error())
			}
			err = consumer.Declare(ch)
			if err != nil {
				log.Errorf(nil, "clear publisher failed, error: %s", err.Error())
			}
			c.consumers[consumer] = ch
		}
	}
}

func (c *Connection) GetChannel() *AMQPChannel {
	return c.channel
}

func (c *Connection) EnsureWithRetry(method func() *message.Error, retry int, errCallback func(*message.Error), reconnectCallback func()) error {
	var lastErr error
	for i := 0; i < retry; i++ {
		if c.IsClosed() {
			return fmt.Errorf("connection %s closed", c.ID)
		}

		lastErr = c.Ensure(method, errCallback, reconnectCallback)
		if lastErr == nil {
			return nil
		}
	}
	if lastErr == nil {
		return nil
	}
	return fmt.Errorf("retry failed, error: %s", lastErr.Error())
}

func (c *Connection) Ensure(method func() *message.Error, errCallback func(*message.Error), reconnectCallback func()) error {
	for {
		if c.IsClosed() {
			return nil
		}

		methodErr := method()
		if methodErr != nil {
			if errCallback != nil {
				errCallback(methodErr)
			}
			c.SetChannel(nil)

			switch methodErr.Code {
			case amqp.ChannelError, amqp.UnexpectedFrame, amqp.InternalError, amqp.FrameError:
				_ = c.amqpConn.RecreateChannel()
			}

			if err := c.amqpConn.EnsureConnection(); err == nil {
				c.SetChannel(c.amqpConn.GetChannel())

				if reconnectCallback != nil {
					reconnectCallback()
				}
			}
		} else {
			return nil
		}
	}
}

func (c *Connection) DeclareConsumer(consumer *Consumer) {
	c.connectionMu.Lock()
	defer c.connectionMu.Unlock()

	_ = c.Ensure(func() *message.Error {
		ch, err := c.CreateChannel()
		if err != nil {
			return &message.Error{
				Code:    amqp.InternalError,
				Reason:  err.Error(),
				Server:  false,
				Recover: false,
			}
		}
		err = consumer.Declare(ch)
		if err != nil {
			log.Debugf(nil, "declare consumer %s failed, error: %s", consumer.ID, err.Error())
			return &message.Error{
				Code:    amqp.InternalError,
				Reason:  err.Error(),
				Server:  false,
				Recover: false,
			}
		}

		c.consumers[consumer] = ch
		return nil
	}, nil, nil)
}

func (c *Connection) DeclareDirectConsumer(topic string, callback func(msg interfaces.Message)) {
	id := uuid.NewString()
	// default exchange
	exchange := ""
	queue := topic
	queueTTL := 0
	queueHa := false

	if queueTTLV, ok := c.properties["amqp_queues_ttl"]; ok {
		queueTTL = queueTTLV.(int)
	}

	if queueHaV, ok := c.properties["amqp_ha_queues"]; ok {
		queueHa = queueHaV.(bool)
	}

	consumer := NewConsumer(amqp.ExchangeTopic, exchange, queue, "", false, false, false, queueHa, queueTTL, callback)
	consumer.ID = id
	c.DeclareConsumer(consumer)
}

func (c *Connection) DeclareTopicConsumer(exchange, topic string, callback func(msg interfaces.Message), queue string) {
	if queue == "" {
		queue = topic
	}
	durable := false
	autoDelete := false
	queueHa := false

	if durableV, ok := c.properties["amqp_durable_queues"]; ok {
		durable = durableV.(bool)
	}

	if autoDeleteV, ok := c.properties["amqp_auto_delete"]; ok {
		autoDelete = autoDeleteV.(bool)
	}

	if queueHaV, ok := c.properties["amqp_ha_queues"]; ok {
		queueHa = queueHaV.(bool)
	}

	consumer := NewConsumer(amqp.ExchangeTopic, exchange, queue, topic, durable, autoDelete, autoDelete, queueHa, 0, callback)
	consumer.ID = uuid.NewString()
	c.DeclareConsumer(consumer)
}

func (c *Connection) DeclareFanoutConsumer(topic string, callback func(msg interfaces.Message)) {
	id := uuid.NewString()
	exchange := fmt.Sprintf("%s_fanout", topic)
	queue := fmt.Sprintf("%s_fanout_%s", topic, id)
	queueTTL := 0
	queueHa := false

	if queueTTLV, ok := c.properties["amqp_queues_ttl"]; ok {
		queueTTL = queueTTLV.(int)
	}

	if queueHaV, ok := c.properties["amqp_ha_queues"]; ok {
		queueHa = queueHaV.(bool)
	}

	consumer := NewConsumer(amqp.ExchangeTopic, exchange, queue, topic, false, true, true, queueHa, queueTTL, callback)
	consumer.ID = id
	c.DeclareConsumer(consumer)
}

func (c *Connection) doHeartbeat() {
	for {
		if c.IsClosed() {
			return
		}

		drainErr := c.amqpConn.DrainEvent(200 * time.Millisecond)
		if drainErr != nil {
			switch drainErr.Code {
			case amqp.ChannelError, amqp.UnexpectedFrame, amqp.InternalError, amqp.FrameError:
				_ = c.amqpConn.RecreateChannel()
				_ = c.amqpConn.EnsureConnection()
			}
		}
	}
}

func (c *Connection) CreateChannel() (*AMQPChannel, error) {
	return c.amqpConn.CreateChannel()
}

type Connector struct {
	Url string

	Heartbeat  time.Duration
	Locale     string
	Properties amqp.Table
}

func NewConnector(connUrl string) (*Connector, error) {
	uri, err := url.Parse(connUrl)
	if err != nil {
		return nil, err
	}

	connector := &Connector{
		Url:       fmt.Sprintf("amqp://%s@%s%s", uri.User.String(), uri.Host, uri.Path),
		Heartbeat: defaultHeartbeat,
		Locale:    defaultLocale,
		Properties: amqp.Table{
			"product": defaultProduct,
			"version": defaultVersion,
		},
	}

	qs := uri.Query()

	heartbeatStr := qs.Get("heartbeat")
	if heartbeatStr != "" {
		heartbeatSec, err := strconv.Atoi(heartbeatStr)
		if err == nil {
			connector.Heartbeat = time.Duration(heartbeatSec) * time.Second
		}
	}

	productStr := qs.Get("product")
	if productStr != "" {
		connector.Properties["product"] = productStr
	}

	versionStr := qs.Get("version")
	if productStr != "" {
		connector.Properties["version"] = versionStr
	}

	queueTTL := qs.Get("amqp_queues_ttl")
	if queueTTL != "" {
		_queueTTL, err := strconv.Atoi(queueTTL)
		if err == nil {
			connector.Properties["amqp_queues_ttl"] = _queueTTL
		}
	}

	queueHA := qs.Get("amqp_ha_queues")
	if queueHA != "" {
		queueHA = strings.ToLower(queueHA)
		if queueHA == "1" || queueHA == "true" {
			connector.Properties["amqp_ha_queues"] = true
		}
	}

	return connector, nil
}

func (c Connector) Open(ctx context.Context, purpose string) (*Connection, error) {
	conn := &Connection{
		ID:         uuid.NewString(),
		url:        c.Url,
		heartbeat:  c.Heartbeat,
		locale:     c.Locale,
		properties: c.Properties,
		consumers:  map[*Consumer]*AMQPChannel{},
		purpose:    purpose,
		connectionMu: ConnectionLock{
			purpose: purpose,
			mu:      sync.Mutex{},
		},
	}
	err := conn.Initialize(ctx)
	if err != nil {
		return nil, err
	}
	return conn, nil
}
