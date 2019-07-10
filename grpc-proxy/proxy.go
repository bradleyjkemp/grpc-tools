package grpc_proxy

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/bradleyjkemp/grpc-tools/internal"
	"github.com/bradleyjkemp/grpc-tools/internal/codec"
	"github.com/bradleyjkemp/grpc-tools/internal/detectcert"
	"github.com/bradleyjkemp/grpc-tools/internal/proxydialer"
	"github.com/bradleyjkemp/grpc-tools/internal/tlsmux"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/http/httpproxy"
	"google.golang.org/grpc"
	_ "google.golang.org/grpc/encoding/gzip"
	"net"
)

type ContextDialer = func(context.Context, string) (net.Conn, error)

type server struct {
	serverOptions []grpc.ServerOption
	grpcServer    *grpc.Server
	logger        logrus.FieldLogger

	port     int
	certFile string
	keyFile  string
	x509Cert *x509.Certificate
	tlsCert  tls.Certificate

	destination string
	connPool    *internal.ConnPool
	dialer      ContextDialer

	listener net.Listener
}

func New(configurators ...Configurator) (*server, error) {
	logger := logrus.New()
	s := &server{
		logger: logger,
		dialer: proxydialer.NewProxyDialer(httpproxy.FromEnvironment().ProxyFunc()),
	}
	s.serverOptions = []grpc.ServerOption{
		grpc.CustomCodec(codec.NoopCodec{}),        // Allows for passing raw []byte messages around
		grpc.UnknownServiceHandler(s.proxyHandler), // All services are unknown so will be proxied
	}

	for _, configurator := range configurators {
		configurator(s)
	}

	// Have to initialise the connpool now because
	// the dialer may been changed by options
	s.connPool = internal.NewConnPool(logger, s.dialer)

	if fLogLevel != "" {
		level, err := logrus.ParseLevel(fLogLevel)
		if err != nil {
			return nil, err
		}
		logger.SetLevel(level)
	}

	if s.certFile == "" && s.keyFile == "" {
		var err error
		s.certFile, s.keyFile, err = detectcert.Detect()
		if err != nil {
			s.logger.WithError(err).Info("Failed to detect certificates")
		}
	}

	if s.certFile != "" && s.keyFile != "" {
		var err error
		s.tlsCert, err = tls.LoadX509KeyPair(s.certFile, s.keyFile)
		if err != nil {
			return nil, err
		}

		s.x509Cert, err = x509.ParseCertificate(s.tlsCert.Certificate[0]) //TODO do we need to parse anything other than [0]?
		if err != nil {
			return nil, err
		}
	}

	return s, nil
}

func (s *server) Start() error {
	var err error
	s.listener, err = net.Listen("tcp", fmt.Sprintf("localhost:%d", s.port))
	if err != nil {
		return fmt.Errorf("failed to listen on port (%d): %v", s.port, err)
	}
	s.logger.Infof("Listening on %s", s.listener.Addr())
	if s.x509Cert != nil {
		s.logger.Infof("Intercepting TLS connections to domains: %s", s.x509Cert.DNSNames)
	} else {
		s.logger.Infof("Not intercepting TLS connections")
	}

	grpcWebHandler := grpcweb.WrapServer(
		grpc.NewServer(s.serverOptions...),
		grpcweb.WithCorsForRegisteredEndpointsOnly(false), // because we are proxying
		grpcweb.WithOriginFunc(func(_ string) bool { return true }),
	)

	proxyLis := newProxyListener(s.logger, s.listener)

	httpServer := newHttpServer(s.logger, grpcWebHandler, proxyLis.internalRedirect, httpReverseProxy)
	httpsServer := withHttpsMiddleware(newHttpServer(s.logger, grpcWebHandler, proxyLis.internalRedirect, httpReverseProxy))

	httpLis, httpsLis := tlsmux.New(s.logger, proxyLis, s.x509Cert, s.tlsCert)

	errChan := make(chan error)
	go func() {
		errChan <- httpServer.Serve(httpLis)
	}()
	go func() {
		// the TLSMux unwraps TLS for us so we use Serve instead of ServeTLS
		errChan <- httpsServer.Serve(httpsLis)
	}()

	return <-errChan
}
