package netutil

import (
	"errors"
	"net"
	"sync"
	"time"
)

var (
	errKeepaliveNotSupported = errors.New("Keepalive not supported")
)

// SharedLimitListener returns a Listener that accepts simultaneous
// connections from the provided Listener only if a shared availability pool
// permits it. Based on https://godoc.org/golang.org/x/net/netutil
func SharedLimitListener(listener net.Listener, limiter *Limiter) net.Listener {
	return &sharedLimitListener{
		Listener: listener,
		limiter:  limiter,
		done:     make(chan struct{}),
	}
}

// Limiter is used to provide a shared pool of connection slots. Use NewLimiter
// to create an instance
type Limiter struct {
	sem chan struct{}
}

// NewLimiter creates a Limiter with the given capacity
func NewLimiter(n int) *Limiter {
	return &Limiter{
		sem: make(chan struct{}, n),
	}
}

type sharedLimitListener struct {
	net.Listener
	closeOnce sync.Once     // ensures the done chan is only closed once
	limiter   *Limiter      // A pool of connection slots shared with other listeners
	done      chan struct{} // no values sent; closed when Close is called
}

// acquire acquires the limiting semaphore. Returns true if successfully
// accquired, false if the listener is closed and the semaphore is not
// acquired.
func (l *sharedLimitListener) acquire() bool {
	select {
	case <-l.done:
		return false
	case l.limiter.sem <- struct{}{}:
		return true
	}
}
func (l *sharedLimitListener) release() { <-l.limiter.sem }

func (l *sharedLimitListener) Accept() (net.Conn, error) {
	acquired := l.acquire()
	// If the semaphore isn't acquired because the listener was closed, expect
	// that this call to accept won't block, but immediately return an error.
	c, err := l.Listener.Accept()
	if err != nil {
		if acquired {
			l.release()
		}
		return nil, err
	}

	// Support TCP Keepalive operations if possible
	tcpConn, _ := c.(*net.TCPConn)

	return &sharedLimitListenerConn{
		Conn:    c,
		tcpConn: tcpConn,
		release: l.release,
	}, nil
}

func (l *sharedLimitListener) Close() error {
	err := l.Listener.Close()
	l.closeOnce.Do(func() { close(l.done) })
	return err
}

type sharedLimitListenerConn struct {
	net.Conn
	tcpConn     *net.TCPConn
	releaseOnce sync.Once
	release     func()
}

func (c *sharedLimitListenerConn) Close() error {
	err := c.Conn.Close()
	c.releaseOnce.Do(c.release)
	return err
}

func (c *sharedLimitListenerConn) SetKeepAlive(enabled bool) error {
	if c.tcpConn == nil {
		return errKeepaliveNotSupported
	}

	return c.tcpConn.SetKeepAlive(enabled)
}

func (c *sharedLimitListenerConn) SetKeepAlivePeriod(period time.Duration) error {
	if c.tcpConn == nil {
		return errKeepaliveNotSupported
	}

	return c.tcpConn.SetKeepAlivePeriod(period)
}
