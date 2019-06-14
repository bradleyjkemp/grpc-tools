package tlsmux

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"github.com/sirupsen/logrus"
	"io"
	"net"
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

func (c tlsMuxConn) OriginalDestination() string {
	switch underlying := c.Conn.(type) {
	case proxiedConnection:
		return underlying.OriginalDestination()
	default:
		return ""
	}
}

func New(logger logrus.FieldLogger, listener net.Listener, cert *x509.Certificate, tlsCert tls.Certificate) (net.Listener, net.Listener) {
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
				handleTlsConn(logger, conn, cert, tlsConns)
			} else {
				nonTlsConns <- conn
			}
		}
	}()
	closer := &sync.Once{}
	nonTlsListener := &tlsMuxListener{
		parent: listener,
		close:  closer,
		conns:  nonTlsConns,
	}
	tlsListener := &tlsMuxListener{
		parent: listener,
		close:  closer,
		conns:  tlsConns,
	}
	return nonTlsListener, tlsListener
}

func handleTlsConn(logger logrus.FieldLogger, conn net.Conn, cert *x509.Certificate, tlsConns chan net.Conn) {
	logger.Debugf("Handling TLS connection %v", conn)
	switch connType := conn.(type) {
	case proxiedConnection:
		if connType.OriginalDestination() == "" {
			logger.Debug("Connection has no original destination so must intercept")
			// cannot be forwarded so must accept regardless of whether we are able to intercept
			tlsConns <- conn
			return
		}
		logger.Debugf("Got TLS connection for destination %s", connType.OriginalDestination())

		// trim the port suffix
		originalHostname := strings.Split(connType.OriginalDestination(), ":")[0]
		if cert != nil && cert.VerifyHostname(originalHostname) == nil {
			// the certificate we have allows us to intercept this connection
			tlsConns <- conn
		} else {
			// cannot intercept so will just transparently proxy instead
			logger.Infof("No certificate able to intercept connections to %s, proxying instead.", originalHostname)
			destConn, err := net.Dial(conn.LocalAddr().Network(), connType.OriginalDestination())
			if err != nil {
				logger.WithError(err).Warnf("Failed proxying connection to %s, Error while dialing.", originalHostname)
				return
			}
			go func() {
				err = forwardConnection(
					conn,
					destConn,
				)
				if err != nil {
					logger.WithError(err).Warnf("Error proxying connection to %s.", originalHostname)
				}
			}()
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
func isTlsConn(conn net.Conn) (net.Conn, bool, error) {
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
	logger logrus.FieldLogger
	net.Listener
	tls bool
}

var (
	httpPeekSize = 8
	// These are the HTTP methods we are interested in handling. Anything else gets bounced.
	httpPattern = regexp.MustCompile(`^(CONNECT)|(POST)|(PRI) `)
)

type proxiedConnection interface {
	OriginalDestination() string
}

func (b nonHTTPBouncer) Accept() (net.Conn, error) {
	for {
		conn, err := b.Listener.Accept()
		if err != nil {
			return nil, err
		}

		switch connType := conn.(type) {
		case proxiedConnection:
			if connType.OriginalDestination() == "" {
				// cannot be forwarded so we must accept the connection
				// regardless of what protocol it speaks.
				return conn, nil
			}

			peeker := bufio.NewReaderSize(conn, httpPeekSize)
			peek, err := peeker.Peek(httpPeekSize)
			if err != nil {
				return nil, err
			}
			b.logger.Debug("peek: ", string(peek))
			if httpPattern.Match(peek) {
				// this is a connection we want to handle
				return tlsMuxConn{
					reader: peeker,
					Conn:   conn,
				}, nil
			}
			b.logger.Debugf("Non HTTP connection to %s detected (peek: %s), proxying instead", connType.OriginalDestination(), string(peek))

			// proxy this connection without interception
			destination := connType.OriginalDestination()
			var destConn net.Conn
			if b.tls {
				destConn, err = tls.Dial(conn.LocalAddr().Network(), destination, nil)
			} else {
				destConn, err = net.Dial(conn.LocalAddr().Network(), destination)
			}
			if err != nil {
				b.logger.WithError(err).Warnf("Error proxying connection to %s.", destination)
			}

			go func() {
				err := forwardConnection(
					conn,
					destConn,
				)
				if err != nil {
					b.logger.WithError(err).Warnf("Error proxying connection to %s.", destination)
				}
			}()

		default:
			// unknown (direct?) connection, must handle it ourselves
			return conn, nil
		}
	}
}
