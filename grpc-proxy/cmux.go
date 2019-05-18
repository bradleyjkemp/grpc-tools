package grpc_proxy

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"regexp"
	"sync"
)

var (
	httpsPattern = regexp.MustCompile(`^\x16\x03[\00-\x03]`) // TLS handshake byte + version number
	peekSize     = 8
)

type chanListener struct {
	parent net.Listener
	close  *sync.Once
	conns  <-chan net.Conn
	errs   <-chan error
}

func (c *chanListener) Accept() (net.Conn, error) {
	select {
	case conn := <-c.conns:
		return conn, nil
	case err := <-c.errs:
		return nil, err
	}
}

func (c *chanListener) Close() error {
	var err error
	c.close.Do(func() {
		err = c.parent.Close()
	})
	return err
}

func (c *chanListener) Addr() net.Addr {
	return c.parent.Addr()
}

type bufferConn struct {
	reader io.Reader
	*net.TCPConn
}

func (c bufferConn) Read(b []byte) (n int, err error) {
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
			if httpsPattern.Match(peek) {
				httpsCon <- bufferConn{
					reader:  peeker,
					TCPConn: conn.(*net.TCPConn),
				}
			} else {
				httpCon <- bufferConn{
					reader:  peeker,
					TCPConn: conn.(*net.TCPConn),
				}
			}
		}
	}()
	closer := &sync.Once{}
	return &chanListener{
			parent: listener,
			close:  closer,
			conns:  httpCon,
		}, &chanListener{
			parent: listener,
			close:  closer,
			conns:  httpsCon,
		}
}
