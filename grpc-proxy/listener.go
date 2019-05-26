package grpc_proxy

import (
	"net"
	"sync"
)

type bidirectionalConn interface {
	net.Conn
	CloseRead() error
	CloseWrite() error
}

type proxiedConn struct {
	bidirectionalConn
	originalDestination string
	tls                 bool
}

// listens on a net.Listener as well as a channel for internal redirects
// while preserving original destination
type proxyListener struct {
	channel chan *proxiedConn
	errs    chan error
	*net.TCPListener
	once sync.Once
}

func (l *proxyListener) Accept() (net.Conn, error) {
	l.once.Do(func() {
		// listen on the actual net.Listener and put into the channel
		go func() {
			l.errs = make(chan error)
			for {
				conn, err := l.TCPListener.AcceptTCP()
				if err != nil {
					l.errs <- err
					continue
				}
				l.channel <- &proxiedConn{
					bidirectionalConn:   conn,
					originalDestination: l.TCPListener.Addr().String(),
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
