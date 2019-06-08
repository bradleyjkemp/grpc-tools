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

// This file implements a listener that splits received connections
// into two listeners depending on whether the connection is (likely)
// a TLS connection. It does this by peeking at the first few bytes
// of the connection and seeing if it looks like a TLS handshake.

var (
	tlsPattern = regexp.MustCompile(`^\x16\x03[\00-\x03]`) // TLS handshake byte + version number
	peekSize   = 3
)

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
	bidirectionalConn
	tls bool
}

func (c tlsMuxConn) Read(b []byte) (n int, err error) {
	return c.reader.Read(b)
}

func newTlsMux(listener net.Listener, cert *x509.Certificate) (nonTls net.Listener, tls net.Listener) {
	var nonTlsConns = make(chan net.Conn, 1)
	var nonTlsErrs = make(chan error, 1)
	var tlsConns = make(chan net.Conn, 1)
	var tlsErrs = make(chan error, 1)
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				nonTlsErrs <- err
				tlsErrs <- err
				continue
			}

			peeker := bufio.NewReaderSize(conn, peekSize)
			peek, err := peeker.Peek(peekSize)
			if err != nil {
				nonTlsErrs <- err
				tlsErrs <- err
			}
			if tlsPattern.Match(peek) {
				handleTlsConn(conn, peeker, cert, tlsConns)
			} else {
				handleNonTlsConn(conn, peeker, nonTlsConns)
			}
		}
	}()
	closer := &sync.Once{}
	return &tlsMuxListener{
			parent: listener,
			close:  closer,
			conns:  nonTlsConns,
		}, &tlsMuxListener{
			parent: listener,
			close:  closer,
			conns:  tlsConns,
		}
}

func handleTlsConn(conn net.Conn, r io.Reader, cert *x509.Certificate, tlsConns chan net.Conn) {
	switch connType := conn.(type) {
	case proxiedConn:
		// trim the port suffix
		originalHostname := strings.Split(connType.originalDestination, ":")[0]
		if cert != nil && cert.VerifyHostname(originalHostname) == nil {
			// the certificate we have allows us to intercept this connection
			tlsConns <- tlsMuxConn{
				reader:            r,
				bidirectionalConn: connType,
				tls:               true,
			}
		} else {
			// cannot intercept so will just transparently proxy instead
			// TODO: move this to a debug log level
			//fmt.Fprintln(os.Stderr, "Err: do not have a certificate that can serve", originalHostname)
			err := forwardConnection(proxiedConn{
				tlsMuxConn{ // TODO: this is pretty messed up but required because of the peeking that has already occurred
					reader:            r,
					bidirectionalConn: connType,
					tls:               true,
				},
				connType.originalDestination,
				true,
			})
			if err != nil {
				fmt.Fprintln(os.Stderr, "Err: error proxying to", originalHostname, err)
			}
		}

	case *net.TCPConn:
		// either this was a direct connection or we're proxying for a hostname we can intercept
		tlsConns <- tlsMuxConn{
			reader:            r,
			bidirectionalConn: connType,
			tls:               true,
		}

	default:
		fmt.Fprintln(os.Stderr, "Err: unknown connection type", connType)
		conn.Close()
	}
}

func handleNonTlsConn(conn net.Conn, r io.Reader, nonTlsConns chan net.Conn) {
	switch bidiConn := conn.(type) {
	case bidirectionalConn:
		nonTlsConns <- tlsMuxConn{
			reader:            r,
			bidirectionalConn: bidiConn,
			tls:               false,
		}
	default:
		fmt.Fprintln(os.Stderr, "Err: unknown connection type", bidiConn)
		conn.Close()
	}

}
