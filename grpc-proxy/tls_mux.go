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
	net.Conn
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

func handleTlsConn(conn tlsMuxConn, cert *x509.Certificate, tlsConns chan net.Conn) {
	switch connType := conn.Conn.(type) {
	case proxiedConn:
		// trim the port suffix
		originalHostname := strings.Split(connType.originalDestination, ":")[0]
		if cert != nil && cert.VerifyHostname(originalHostname) == nil {
			// the certificate we have allows us to intercept this connection
			tlsConns <- conn
		} else {
			// cannot intercept so will just transparently proxy instead
			// TODO: move this to a debug log level
			//fmt.Fprintln(os.Stderr, "Err: do not have a certificate that can serve", originalHostname)
			err := forwardConnection(
				conn,
				connType.originalDestination,
			)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Err: error proxying to", originalHostname, err)
			}
		}

	case *net.TCPConn:
		// this was a direct connection (proxy is being used in fallback mode)
		tlsConns <- conn

	default:
		fmt.Fprintln(os.Stderr, "Err: unknown connection type", connType)
		conn.Close()
	}
}

// Takes a net.Conn and peeks to see if it is a TLS conn or not.
// The original net.Conn *cannot* be used after calling.
func isTlsConn(conn net.Conn) (tlsMuxConn, bool, error) {
	peeker := bufio.NewReaderSize(conn, peekSize)
	peek, err := peeker.Peek(peekSize)
	if err != nil {
		return tlsMuxConn{}, false, err
	}

	return tlsMuxConn{
		reader: peeker,
		Conn:   conn,
	}, tlsPattern.Match(peek), nil
}
