package grpc_proxy

import (
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
	channel chan proxiedConn
	errs    chan error
	*net.TCPListener
	once sync.Once
}

func newProxyListener(listener *net.TCPListener) *proxyListener {
	return &proxyListener{
		channel:     make(chan proxiedConn),
		errs:        make(chan error),
		TCPListener: listener,
		once:        sync.Once{},
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
				conn, err := l.TCPListener.AcceptTCP()
				if err != nil {
					l.errs <- err
					continue
				}
				l.channel <- proxiedConn{
					Conn:         conn,
					originalDest: l.TCPListener.Addr().String(),
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
