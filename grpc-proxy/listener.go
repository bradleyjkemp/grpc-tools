package grpc_proxy

import (
	"github.com/sirupsen/logrus"
	"net"
	"sync"
)

type proxiedConn struct {
	net.Conn
	originalDest string
}

func (p proxiedConn) originalDestination() string {
	return p.originalDest
}

// listens on a net.Listener as well as a channel for internal redirects
// while preserving original destination
type proxyListener struct {
	logger  *logrus.Logger
	channel chan proxiedConn
	errs    chan error
	net.Listener
	once sync.Once
}

func newProxyListener(logger *logrus.Logger, listener net.Listener) *proxyListener {
	return &proxyListener{
		logger:   logger,
		channel:  make(chan proxiedConn),
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
				l.channel <- proxiedConn{
					Conn:         conn,
					originalDest: l.Listener.Addr().String(),
				}
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
