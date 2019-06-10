package grpc_proxy

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"os"
	"regexp"
	"strings"
	"sync"
)

// This file implements a listener that splits received connections
// into two listeners depending on whether the connection is (likely)
// a TLS connection. It does this by peeking at the first few bytes
// of the connection and seeing if it looks like a TLS handshake.

type tlsMuxListener struct {
	parent net.Listener
	close  *sync.Once
	conns  <-chan net.Conn
	errs   <-chan error
}

func (c *tlsMuxListener) Accept() (net.Conn, error) {
	select {
	case conn := <-c.conns:
		return conn, nil
	case err := <-c.errs:
		return nil, err
	}
}

func (c *tlsMuxListener) Close() error {
	var err error
	c.close.Do(func() {
		err = c.parent.Close()
	})
	return err
}

func (c *tlsMuxListener) Addr() net.Addr {
	return c.parent.Addr()
}

type tlsMuxConn struct {
	reader io.Reader
	net.Conn
}

func (c tlsMuxConn) Read(b []byte) (n int, err error) {
	return c.reader.Read(b)
}

func (c tlsMuxConn) originalDestination() string {
	switch underlying := c.Conn.(type) {
	case proxiedConnection:
		return underlying.originalDestination()
	default:
		return ""
	}
}

func newTlsMux(listener net.Listener, cert *x509.Certificate, tlsCert tls.Certificate) (net.Listener, net.Listener) {
	var nonTlsConns = make(chan net.Conn, 1)
	var nonTlsErrs = make(chan error, 1)
	var tlsConns = make(chan net.Conn, 1)
	var tlsErrs = make(chan error, 1)
	go func() {
		for {
			rawConn, err := listener.Accept()
			if err != nil {
				nonTlsErrs <- err
				tlsErrs <- err
				continue
			}

			conn, isTls, err := isTlsConn(rawConn)

			if isTls {
				handleTlsConn(conn, cert, tlsConns)
			} else {
				nonTlsConns <- conn
			}
		}
	}()
	closer := &sync.Once{}
	return nonHTTPBouncer{&tlsMuxListener{
			parent: listener,
			close:  closer,
			conns:  nonTlsConns,
		}}, nonHTTPBouncer{tls.NewListener(&tlsMuxListener{
			parent: listener,
			close:  closer,
			conns:  tlsConns,
		}, &tls.Config{
			Certificates: []tls.Certificate{tlsCert},
		})}
}

func handleTlsConn(conn tlsMuxConn, cert *x509.Certificate, tlsConns chan net.Conn) {
	switch connType := conn.Conn.(type) {
	case proxiedConnection:
		if connType.originalDestination() == "" {
			// cannot be forwarded so must accept regardless of whether we are able to intercept
			tlsConns <- conn
			return
		}

		// trim the port suffix
		originalHostname := strings.Split(connType.originalDestination(), ":")[0]
		if cert != nil && cert.VerifyHostname(originalHostname) == nil {
			// the certificate we have allows us to intercept this connection
			tlsConns <- conn
		} else {
			// cannot intercept so will just transparently proxy instead
			// TODO: move this to a debug log level
			//fmt.Fprintln(os.Stderr, "Err: do not have a certificate that can serve", originalHostname)
			destConn, err := net.Dial(conn.LocalAddr().Network(), connType.originalDestination())
			if err != nil {
				fmt.Fprintln(os.Stderr, "Err: error proxying to", originalHostname, err)
			}
			err = forwardConnection(
				conn,
				destConn,
			)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Err: error proxying to", originalHostname, err)
			}
		}

	default:
		tlsConns <- conn
	}
}

var (
	tlsPattern  = regexp.MustCompile(`^\x16\x03[\00-\x03]`) // TLS handshake byte + version number
	tlsPeekSize = 3
)

// Takes a net.Conn and peeks to see if it is a TLS conn or not.
// The original net.Conn *cannot* be used after calling.
func isTlsConn(conn net.Conn) (tlsMuxConn, bool, error) {
	peeker := bufio.NewReaderSize(conn, tlsPeekSize)
	peek, err := peeker.Peek(tlsPeekSize)
	if err != nil {
		return tlsMuxConn{}, false, err
	}

	return tlsMuxConn{
		reader: peeker,
		Conn:   conn,
	}, tlsPattern.Match(peek), nil
}

// nonHTTPBouncer wraps a net.Listener and detects whether or not
// the connection is HTTP. If not then it proxies the connection
// to the original destination.
// This is a single purpose version of github.com/soheilhy/cmux
type nonHTTPBouncer struct {
	net.Listener
}

var (
	httpPeekSize = 8
	// These are the HTTP methods we are interested in handling. Anything else gets bounced.
	httpPattern = regexp.MustCompile(`^(CONNECT)|(POST)|(PRI) `)
)

type proxiedConnection interface {
	originalDestination() string
}

func (b nonHTTPBouncer) Accept() (net.Conn, error) {
	for {
		conn, err := b.Listener.Accept()
		if err != nil {
			return nil, err
		}

		switch connType := conn.(type) {
		case proxiedConnection:
			if connType.originalDestination() == "" {
				// cannot be forwarded so we must accept the connection
				// regardless of what protocol it speaks.
				return conn, nil
			}

			peeker := bufio.NewReaderSize(conn, httpPeekSize)
			peek, err := peeker.Peek(httpPeekSize)
			if err != nil {
				return nil, err
			}
			if httpPattern.Match(peek) {
				// this is a connection we want to handle
				return tlsMuxConn{
					reader: peeker,
					Conn:   conn,
				}, nil
			}

			// proxy this connection without interception
			destination := connType.originalDestination()
			destConn, err := tls.Dial(conn.LocalAddr().Network(), destination, nil)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Err: error proxying to", destination, err)
			}

			go func() {
				err := forwardConnection(
					conn,
					destConn,
				)
				if err != nil {
					fmt.Fprintln(os.Stderr, "Err: error proxying to", destination, err)
				}
			}()

		default:
			// unknown (direct?) connection, must handle it ourselves
			return conn, nil
		}
	}
}
