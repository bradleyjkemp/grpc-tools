package tlsmux

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/bradleyjkemp/grpc-tools/internal/peekconn"
	"github.com/sirupsen/logrus"
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
	net.Listener
	close *sync.Once
	conns <-chan net.Conn
	errs  <-chan error
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
		err = c.Listener.Close()
	})
	return err
}

func New(logger logrus.FieldLogger, listener net.Listener, cert *x509.Certificate, tlsCert tls.Certificate) (net.Listener, net.Listener) {
	var nonTlsConns = make(chan net.Conn, 128) // TODO decide on good buffer sizes for these channels
	var nonTlsErrs = make(chan error, 128)
	var tlsConns = make(chan net.Conn, 128)
	var tlsErrs = make(chan error, 128)
	go func() {
		for {
			rawConn, err := listener.Accept()
			if err != nil {
				nonTlsErrs <- err
				tlsErrs <- err
				continue
			}

			go func() {
				conn := peekconn.New(rawConn)

				isTls, err := conn.PeekMatch(tlsPattern, tlsPeekSize)
				if err != nil {
					nonTlsErrs <- err
					tlsErrs <- err
				}
				if isTls {
					handleTlsConn(logger, conn, cert, tlsConns)
				} else {
					nonTlsConns <- conn
				}
			}()

		}
	}()

	closer := &sync.Once{}
	nonTlsListener := nonHTTPBouncer{
		logger,
		&tlsMuxListener{
			Listener: listener,
			close:    closer,
			conns:    nonTlsConns,
		},
		false,
	}
	tlsListener := nonHTTPBouncer{
		logger,
		tls.NewListener(&tlsMuxListener{
			Listener: listener,
			close:    closer,
			conns:    tlsConns,
		}, &tls.Config{
			Certificates: []tls.Certificate{tlsCert},
		}),
		true,
	}
	return nonTlsListener, tlsListener
}

func handleTlsConn(logger logrus.FieldLogger, conn net.Conn, cert *x509.Certificate, tlsConns chan net.Conn) {
	logger.Debugf("Handling TLS connection %v", conn)

	proxConn, ok := conn.(proxiedConnection)
	if !ok {
		tlsConns <- conn
		return
	}

	if proxConn.OriginalDestination() == "" {
		logger.Debug("Connection has no original destination so must intercept")
		// cannot be forwarded so must accept regardless of whether we are able to intercept
		tlsConns <- conn
		return
	}

	logger.Debugf("Got TLS connection for destination %s", proxConn.OriginalDestination())

	// trim the port suffix
	originalHostname := strings.Split(proxConn.OriginalDestination(), ":")[0]
	if cert != nil && cert.VerifyHostname(originalHostname) == nil {
		// the certificate we have allows us to intercept this connection
		tlsConns <- conn
		return
	}

	// cannot intercept so will just transparently proxy instead
	logger.Debugf("No certificate able to intercept connections to %s, proxying instead.", originalHostname)
	destConn, err := net.Dial(conn.LocalAddr().Network(), proxConn.OriginalDestination())
	if err != nil {
		logger.WithError(err).Warnf("Failed proxying connection to %s, Error while dialing.", originalHostname)
		return
	}
	err = forwardConnection(
		conn,
		destConn,
	)
	if err != nil {
		logger.WithError(err).Warnf("Error proxying connection to %s.", originalHostname)
	}
}

var (
	tlsPattern  = regexp.MustCompile(`^\x16\x03[\x00-\x03]`) // TLS handshake byte + version number
	tlsPeekSize = 3
)

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
	conn, err := b.Listener.Accept()
	if err != nil {
		return nil, err
	}

	proxConn, ok := conn.(proxiedConnection)
	if !ok || proxConn.OriginalDestination() == "" {
		// unknown (direct?) connection, must handle it ourselves
		return conn, nil
	}

	peekedConn := peekconn.New(conn)
	match, err := peekedConn.PeekMatch(httpPattern, httpPeekSize)
	if err != nil {
		return nil, err
	}
	if match {
		// this is a connection we want to handle
		return peekedConn, nil
	}
	b.logger.Debugf("Bouncing non-HTTP connection to destination %s", proxConn.OriginalDestination())

	// proxy this connection without interception
	go func() {
		destination := proxConn.OriginalDestination()
		var destConn net.Conn
		if b.tls {
			destConn, err = tls.Dial(conn.LocalAddr().Network(), destination, nil)
		} else {
			destConn, err = net.Dial(conn.LocalAddr().Network(), destination)
		}
		if err != nil {
			b.logger.WithError(err).Warnf("Error proxying connection to %s.", destination)
			return
		}

		err := forwardConnection(
			peekedConn,
			destConn,
		)
		if err != nil {
			b.logger.WithError(err).Warnf("Error proxying connection to %s.", destination)
		}
	}()

	return b.Accept()
}
