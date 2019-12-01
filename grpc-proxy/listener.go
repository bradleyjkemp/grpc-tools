package grpc_proxy

import (
	"net"
	"sync"

	"github.com/sirupsen/logrus"
)

type proxiedConn struct {
	net.Conn
	originalDest string
}

func (p proxiedConn) OriginalDestination() string {
	return p.originalDest
}

// listens on a net.Listener as well as a channel for internal redirects
// while preserving original destination
type proxyListener struct {
	logger  logrus.FieldLogger
	channel chan net.Conn
	errs    chan error
	net.Listener
	once sync.Once
}

func newProxyListener(logger logrus.FieldLogger, listener net.Listener) *proxyListener {
	return &proxyListener{
		logger:   logger,
		channel:  make(chan net.Conn),
		errs:     make(chan error),
		Listener: listener,
		once:     sync.Once{},
	}
}

func (l *proxyListener) internalRedirect(conn net.Conn, originalDestination string) {
	l.channel <- proxiedConn{conn, originalDestination}
}

func (l *proxyListener) Accept() (net.Conn, error) {
	l.once.Do(func() {
		// listen on the actual net.Listener and put into the channel
		go func() {
			for {
				conn, err := l.Listener.Accept()
				if err != nil {
					l.errs <- err
					continue
				}
				l.logger.Debugf("Got connection from address %v", conn.RemoteAddr())
				l.channel <- conn
			}
		}()
	})

	select {
	case conn := <-l.channel:
		return conn, nil
	case err := <-l.errs:
		return nil, err
	}
}
