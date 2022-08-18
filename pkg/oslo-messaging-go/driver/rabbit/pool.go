package rabbit

import (
	"context"
	"errors"
	"log"
	"seraph/pkg/oslo-messaging-go/interfaces"
	"seraph/pkg/oslo-messaging-go/message"
	"sync"
	"sync/atomic"
	"time"
)

var (
	// ErrClosed performs any operation on the closed client will return this error.
	ErrClosed = errors.New("client is closed")

	// ErrPoolTimeout timed out waiting to get a connection from the connection pool.
	ErrPoolTimeout = errors.New("connection pool timeout")
)

var poolTimers = sync.Pool{
	New: func() interface{} {
		t := time.NewTimer(time.Hour)
		t.Stop()
		return t
	},
}

type lastDialErrorWrap struct {
	err error
}

// PoolStats contains pool state information and accumulated stats.
type PoolStats struct {
	Hits     uint32 // number of times free connection was found in the pool
	Misses   uint32 // number of times free connection was NOT found in the pool
	Timeouts uint32 // number of times a wait timeout occurred

	TotalConns uint32 // number of total connections in the pool
	IdleConns  uint32 // number of idle connections in the pool
	StaleConns uint32 // number of stale connections removed from the pool
}

type ConnPool struct {
	maxCount           int
	idleTimeout        time.Duration
	idleCheckFrequency time.Duration
	minIdleCount       int
	maxIdleCount       int
	poolTimeout        time.Duration

	connector     Connector
	dialErrorsNum uint32 // atomic
	lastDialError atomic.Value

	queueCh chan struct{}

	connsMu   sync.Mutex
	conns     []*Connection
	idleConns []*Connection
	poolSize  int
	idleCount int

	stats PoolStats

	_closed  uint32 // atomic
	closedCh chan struct{}
}

func NewPool(connector Connector) *ConnPool {
	p := &ConnPool{
		connector:          connector,
		closedCh:           make(chan struct{}),
		idleCheckFrequency: 5 * time.Second,
		maxCount:           10,
		minIdleCount:       0,
		poolTimeout:        60 * time.Second,
	}
	_ = p.Initialize()
	return p
}

func (p *ConnPool) SetPoolTimeout(timeout time.Duration) {
	p.poolTimeout = timeout
}

func (p *ConnPool) SetMaxCount(count int) {
	p.maxCount = count
}

func (p *ConnPool) SetMaxIdleCount(count int) {
	p.maxIdleCount = count
}

func (p *ConnPool) SetMinIdleCount(count int) {
	p.minIdleCount = count
}

func (p *ConnPool) SetIdleTimeout(timeout time.Duration) {
	p.idleTimeout = timeout
}

func (p *ConnPool) Initialize() error {
	p.queueCh = make(chan struct{}, p.maxCount)
	p.conns = make([]*Connection, 0)
	p.idleConns = make([]*Connection, 0)

	p.connsMu.Lock()
	p.checkMinIdleCount()
	p.connsMu.Unlock()

	if p.idleTimeout > 0 && p.idleCheckFrequency > 0 {
		go p.reaper(p.idleCheckFrequency)
	}
	return nil
}

func (p *ConnPool) checkMinIdleCount() {
	if p.minIdleCount == 0 {
		return
	}
	for p.poolSize < p.maxCount && p.idleCount < p.minIdleCount {
		p.poolSize++
		p.idleCount++

		go func() {
			err := p.addIdleConn()
			if err != nil && err != ErrClosed {
				p.connsMu.Lock()
				p.poolSize--
				p.idleCount--
				p.connsMu.Unlock()
			}
		}()
	}
}

func (p *ConnPool) addIdleConn() error {
	cn, err := p.dialConn(context.TODO(), true)
	if err != nil {
		return err
	}

	p.connsMu.Lock()
	defer p.connsMu.Unlock()

	// It is not allowed to add new connections to the closed connection pool.
	if p.closed() {
		_ = cn.Close()
		return ErrClosed
	}

	p.conns = append(p.conns, cn)
	p.idleConns = append(p.idleConns, cn)
	return nil
}

func (p *ConnPool) NewConn(ctx context.Context) (*Connection, error) {
	return p.newConn(ctx, false)
}

func (p *ConnPool) newConn(ctx context.Context, pooled bool) (*Connection, error) {
	cn, err := p.dialConn(ctx, pooled)
	if err != nil {
		return nil, err
	}

	p.connsMu.Lock()
	defer p.connsMu.Unlock()

	// It is not allowed to add new connections to the closed connection pool.
	if p.closed() {
		_ = cn.Close()
		return nil, ErrClosed
	}

	p.conns = append(p.conns, cn)
	if pooled {
		// If pool is full remove the cn on next Put.
		if p.poolSize >= p.maxCount {
			cn.pooled = false
		} else {
			p.poolSize++
		}
	}

	return cn, nil
}

func (p *ConnPool) dial(ctx context.Context, purpose string) (*Connection, error) {
	return p.connector.Open(ctx, purpose)
}

func (p *ConnPool) dialConn(ctx context.Context, pooled bool) (*Connection, error) {
	if p.closed() {
		return nil, ErrClosed
	}

	if atomic.LoadUint32(&p.dialErrorsNum) >= uint32(p.maxCount) {
		return nil, p.getLastDialError()
	}

	conn, err := p.dial(ctx, message.PurposeSend)
	if err != nil {
		p.setLastDialError(err)
		if atomic.AddUint32(&p.dialErrorsNum, 1) == uint32(p.maxCount) {
			go p.tryDial()
		}
		return nil, err
	}

	conn.pooled = pooled
	conn.pool = p
	return conn, nil
}

func (p *ConnPool) tryDial() {
	for {
		if p.closed() {
			return
		}

		conn, err := p.dial(context.Background(), message.PurposeSend)
		if err != nil {
			p.setLastDialError(err)
			time.Sleep(time.Second)
			continue
		}

		atomic.StoreUint32(&p.dialErrorsNum, 0)
		_ = conn.Close()
		return
	}
}

func (p *ConnPool) setLastDialError(err error) {
	p.lastDialError.Store(&lastDialErrorWrap{err: err})
}

func (p *ConnPool) getLastDialError() error {
	err, _ := p.lastDialError.Load().(*lastDialErrorWrap)
	if err != nil {
		return err.err
	}
	return nil
}

// Get returns existed connection from the pool or creates a new one.
func (p *ConnPool) Get(ctx context.Context) (*Connection, error) {
	if p.closed() {
		return nil, ErrClosed
	}

	if err := p.waitTurn(ctx); err != nil {
		return nil, err
	}

	for {
		p.connsMu.Lock()
		cn, err := p.popIdle()
		p.connsMu.Unlock()

		if err != nil {
			return nil, err
		}

		if cn == nil {
			break
		}

		if p.isStaleConn(cn) {
			_ = p.CloseConn(cn)
			continue
		}

		atomic.AddUint32(&p.stats.Hits, 1)
		cn.usedAt = time.Now()
		return cn, nil
	}

	atomic.AddUint32(&p.stats.Misses, 1)

	newcn, err := p.newConn(ctx, true)
	if err != nil {
		p.freeTurn()
		return nil, err
	}

	return newcn, nil
}

func (p *ConnPool) getTurn() {
	p.queueCh <- struct{}{}
}

func (p *ConnPool) waitTurn(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	select {
	case p.queueCh <- struct{}{}:
		return nil
	default:
	}

	timer := poolTimers.Get().(*time.Timer)
	timer.Reset(p.poolTimeout)

	select {
	case <-ctx.Done():
		if !timer.Stop() {
			<-timer.C
		}
		poolTimers.Put(timer)
		return ctx.Err()
	case p.queueCh <- struct{}{}:
		if !timer.Stop() {
			<-timer.C
		}
		poolTimers.Put(timer)
		return nil
	case <-timer.C:
		poolTimers.Put(timer)
		atomic.AddUint32(&p.stats.Timeouts, 1)
		return ErrPoolTimeout
	}
}

func (p *ConnPool) freeTurn() {
	<-p.queueCh
}

func (p *ConnPool) popIdle() (*Connection, error) {
	if p.closed() {
		return nil, ErrClosed
	}
	n := len(p.idleConns)
	if n == 0 {
		return nil, nil
	}

	cn := p.idleConns[0]
	copy(p.idleConns, p.idleConns[1:])
	p.idleConns = p.idleConns[:n-1]
	p.idleCount--
	p.checkMinIdleCount()
	return cn, nil
}

func (p *ConnPool) Put(cn *Connection) {
	if !cn.pooled {
		p.Remove(cn, nil)
		return
	}

	p.connsMu.Lock()
	p.idleConns = append(p.idleConns, cn)
	p.idleCount++
	p.connsMu.Unlock()
	p.freeTurn()
}

func (p *ConnPool) Remove(cn *Connection, reason error) {
	p.removeConnWithLock(cn)
	p.freeTurn()
	_ = p.closeConn(cn)
}

func (p *ConnPool) CloseConn(cn *Connection) error {
	p.removeConnWithLock(cn)
	return p.closeConn(cn)
}

func (p *ConnPool) removeConnWithLock(cn *Connection) {
	p.connsMu.Lock()
	p.removeConn(cn)
	p.connsMu.Unlock()
}

func (p *ConnPool) removeConn(cn *Connection) {
	for i, c := range p.conns {
		if c == cn {
			p.conns = append(p.conns[:i], p.conns[i+1:]...)
			if cn.pooled {
				p.poolSize--
				p.checkMinIdleCount()
			}
			return
		}
	}
}

func (p *ConnPool) closeConn(cn *Connection) error {
	return cn.Close()
}

// Len returns total number of connections.
func (p *ConnPool) Len() int {
	p.connsMu.Lock()
	n := len(p.conns)
	p.connsMu.Unlock()
	return n
}

// IdleLen returns number of idle connections.
func (p *ConnPool) IdleLen() int {
	p.connsMu.Lock()
	n := p.idleCount
	p.connsMu.Unlock()
	return n
}

func (p *ConnPool) Stats() *PoolStats {
	idleLen := p.IdleLen()
	return &PoolStats{
		Hits:     atomic.LoadUint32(&p.stats.Hits),
		Misses:   atomic.LoadUint32(&p.stats.Misses),
		Timeouts: atomic.LoadUint32(&p.stats.Timeouts),

		TotalConns: uint32(p.Len()),
		IdleConns:  uint32(idleLen),
		StaleConns: atomic.LoadUint32(&p.stats.StaleConns),
	}
}

func (p *ConnPool) closed() bool {
	return atomic.LoadUint32(&p._closed) == 1
}

func (p *ConnPool) Filter(fn func(*Connection) bool) error {
	p.connsMu.Lock()
	defer p.connsMu.Unlock()

	var firstErr error
	for _, cn := range p.conns {
		if fn(cn) {
			if err := p.closeConn(cn); err != nil && firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

func (p *ConnPool) Close() error {
	if !atomic.CompareAndSwapUint32(&p._closed, 0, 1) {
		return ErrClosed
	}
	close(p.closedCh)

	var firstErr error
	p.connsMu.Lock()
	for _, cn := range p.conns {
		if err := p.closeConn(cn); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	p.conns = nil
	p.poolSize = 0
	p.idleConns = nil
	p.idleCount = 0
	p.connsMu.Unlock()

	return firstErr
}

func (p *ConnPool) reaper(frequency time.Duration) {
	ticker := time.NewTicker(frequency)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// It is possible that ticker and closedCh arrive together,
			// and select pseudo-randomly pick ticker case, we double-check
			// here to prevent being executed after closed.
			if p.closed() {
				return
			}
			_, err := p.ReapStaleConns()
			if err != nil {
				log.Printf("ReapStaleConns failed: %s", err)
				continue
			}
		case <-p.closedCh:
			return
		}
	}
}

func (p *ConnPool) ReapStaleConns() (int, error) {
	var n int
	for {
		p.getTurn()

		p.connsMu.Lock()
		cn := p.reapStaleConn()
		p.connsMu.Unlock()

		p.freeTurn()

		if cn != nil {
			_ = p.closeConn(cn)
			n++
		} else {
			break
		}
	}
	atomic.AddUint32(&p.stats.StaleConns, uint32(n))
	return n, nil
}

func (p *ConnPool) reapStaleConn() *Connection {
	if len(p.idleConns) == 0 {
		return nil
	}

	cn := p.idleConns[0]
	if !p.isStaleConn(cn) {
		return nil
	}

	p.idleConns = append(p.idleConns[:0], p.idleConns[1:]...)
	p.idleCount--
	p.removeConn(cn)

	return cn
}

func (p *ConnPool) isStaleConn(cn *Connection) bool {
	if p.idleTimeout == 0 {
		return false
	}

	now := time.Now()
	if p.idleTimeout > 0 && now.Sub(cn.usedAt) >= p.idleTimeout {
		return true
	}
	return false
}

func (p *ConnPool) GetConnection(ctx context.Context, purpose string) (interfaces.Conn, error) {
	if purpose == message.PurposeSend {
		if conn, err := p.Get(ctx); err != nil {
			return nil, err
		} else {
			conn.purpose = purpose
			return conn, nil
		}
	} else {
		if conn, err := p.dial(ctx, purpose); err != nil {
			return nil, err
		} else {
			return conn, nil
		}
	}
}

func (p *ConnPool) WithConnection(ctx context.Context, purpose string, callback func(conn interfaces.Conn) (interface{}, error)) (interface{}, error) {
	conn, err := p.GetConnection(ctx, purpose)
	if err != nil {
		return nil, err
	}
	defer func() {
		realConn := conn.(*Connection)
		if realConn.purpose == message.PurposeSend {
			p.Put(realConn)
		} else {
			conn.Release()
		}
	}()
	return callback(conn)
}
