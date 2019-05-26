package grpc_proxy

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/bradleyjkemp/grpc-tools/internal"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"net"
	"os"
)

type server struct {
	serverOptions []grpc.ServerOption
	grpcServer    *grpc.Server

	port     int
	certFile string
	keyFile  string
	x509Cert *x509.Certificate

	destination string
	connPool    *internal.ConnPool

	listener net.Listener
}

func New(configurators ...Configurator) (*server, error) {
	s := &server{
		connPool: internal.NewConnPool(),
	}
	s.serverOptions = []grpc.ServerOption{
		grpc.CustomCodec(NoopCodec{}),              // Allows for passing raw []byte messages around
		grpc.UnknownServiceHandler(s.proxyHandler), // All services are unknown so will be proxied
	}

	for _, configurator := range configurators {
		configurator(s)
	}

	if s.certFile != "" && s.keyFile != "" {
		tlsCert, err := tls.LoadX509KeyPair(s.certFile, s.keyFile)
		if err != nil {
			return nil, err
		}

		s.x509Cert, err = x509.ParseCertificate(tlsCert.Certificate[0]) //TODO do we need to parse anything other than [0]?
		if err != nil {
			return nil, err
		}
	}

	return s, nil
}

func (s *server) Start() error {
	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", s.port))
	if err != nil {
		return fmt.Errorf("failed to listen on port (%d): %v", s.port, err)
	}
	fmt.Fprintf(os.Stderr, "Listening on %s\n", listener.Addr()) // TODO move this to a logger controllable by options

	grpcWebHandler := grpcweb.WrapServer(
		grpc.NewServer(s.serverOptions...),
		grpcweb.WithCorsForRegisteredEndpointsOnly(false), // because we are proxying
	)

	proxyLis := newProxyListener(listener.(*net.TCPListener))

	httpServer := newHttpServer(grpcWebHandler, proxyLis.internalRedirect)
	httpServer.Handler = h2c.NewHandler(httpServer.Handler, &http2.Server{}) // Adds support for unencrypted HTTP
	httpsServer := withHttpsMarkerMiddleware(newHttpServer(grpcWebHandler, proxyLis.internalRedirect))

	httpLis, httpsLis := newTlsMux(proxyLis, s.x509Cert)
	go httpsServer.ServeTLS(httpsLis, s.certFile, s.keyFile)

	return httpServer.Serve(httpLis)
}
