package grpc_proxy

import (
	"net"
	"sync"
)

type proxiedConn struct {
	net.Conn
	originalDestination string
}

// listens on a net.Listener as well as a channel for internal redirects
// while preserving original destination
type proxyListener struct {
	channel chan *proxiedConn
	errs    chan error
	net.Listener
	once sync.Once
}

func (l *proxyListener) Accept() (net.Conn, error) {
	l.once.Do(func() {
		// listen on the actual net.Listener and put into the channel
		go func() {
			l.errs = make(chan error)
			for {
				conn, err := l.Listener.Accept()
				if err != nil {
					l.errs <- err
					continue
				}
				l.channel <- &proxiedConn{
					Conn:                conn,
					originalDestination: l.Listener.Addr().String(),
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
