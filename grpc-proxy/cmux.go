package grpc_proxy

import (
	"bufio"
	"io"
	"net"
	"regexp"
	"sync"
)

var (
	tlsPattern = regexp.MustCompile(`^\x16\x03[\00-\x03]`) // TLS handshake byte + version number
	peekSize   = 3
)

type cmuxListener struct {
	parent net.Listener
	close  *sync.Once
	conns  <-chan net.Conn
	errs   <-chan error
}

func (c *cmuxListener) Accept() (net.Conn, error) {
	select {
	case conn := <-c.conns:
		return conn, nil
	case err := <-c.errs:
		return nil, err
	}
}

func (c *cmuxListener) Close() error {
	var err error
	c.close.Do(func() {
		err = c.parent.Close()
	})
	return err
}

func (c *cmuxListener) Addr() net.Addr {
	return c.parent.Addr()
}

type cmuxConn struct {
	reader io.Reader
	net.Conn
}

func (c cmuxConn) Read(b []byte) (n int, err error) {
	return c.reader.Read(b)
}

func newHttpHttpsMux(listener net.Listener) (net.Listener, net.Listener) {
	var httpCon = make(chan net.Conn, 1)
	var httpErr = make(chan error, 1)
	var httpsCon = make(chan net.Conn, 1)
	var httpsErr = make(chan error, 1)
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				httpErr <- err
				httpsErr <- err
				continue
			}

			peeker := bufio.NewReaderSize(conn, peekSize)
			peek, err := peeker.Peek(peekSize)
			if err != nil {
				httpErr <- err
				httpsErr <- err
			}
			if tlsPattern.Match(peek) {
				httpsCon <- cmuxConn{
					reader: peeker,
					Conn:   conn,
				}
			} else {
				httpCon <- cmuxConn{
					reader: peeker,
					Conn:   conn,
				}
			}
		}
	}()
	closer := &sync.Once{}
	return &cmuxListener{
			parent: listener,
			close:  closer,
			conns:  httpCon,
		}, &cmuxListener{
			parent: listener,
			close:  closer,
			conns:  httpsCon,
		}
}
