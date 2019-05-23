package grpc_proxy

import (
	"bufio"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"os"
	"regexp"
	"strings"
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
	tls bool
}

func (c cmuxConn) Read(b []byte) (n int, err error) {
	return c.reader.Read(b)
}

func newHttpHttpsMux(listener net.Listener, cert *x509.Certificate) (net.Listener, net.Listener) {
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
				if cert == nil {
					// TODO: don't kill the connection here: proxy it without interception to the real host
					fmt.Println("Err: received tls connection but no certificate set up")
					conn.Close()
					continue
				}

				if proxConn, ok := conn.(*proxiedConn); ok {
					// trim the port suffix
					originalHostname := strings.Split(proxConn.originalDestination, ":")[0]
					if cert.VerifyHostname(originalHostname) != nil {
						fmt.Fprintln(os.Stderr, "Err: do not have a certificate that can serve", originalHostname)
						// TODO: don't kill the connection here: proxy it without interception to the real host
						proxConn.Close()
						continue
					}
				}

				// either this was a direct connection or we're proxying for a hostname we can intercept
				httpsCon <- cmuxConn{
					reader: peeker,
					Conn:   conn,
					tls:    true,
				}
			} else {
				httpCon <- cmuxConn{
					reader: peeker,
					Conn:   conn,
					tls:    false,
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
