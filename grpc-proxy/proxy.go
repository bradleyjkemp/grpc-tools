package grpc_proxy

import (
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"net"
	"net/http"
)

var proxyStreamDesc = &grpc.StreamDesc{
	ServerStreams: true,
	ClientStreams: true,
}

type Server interface{}

type server struct {
	err              error // filled if any errors occur during startup
	destinationCreds credentials.TransportCredentials
	serverOptions    []grpc.ServerOption
	grpcServer       *grpc.Server
	destination      *grpc.ClientConn
	interceptor      grpc.StreamServerInterceptor
	certFile         string
	keyFile          string
	grpcWeb          bool
}

func New(configurators ...Configurator) (*server, error) {
	s := &server{}
	for _, configurator := range configurators {
		configurator(s)
	}

	if s.err != nil {
		return nil, s.err
	}

	serverOptions := append(s.serverOptions,
		grpc.CustomCodec(NoopCodec{}),
		grpc.UnknownServiceHandler(s.proxyHandler))

	if s.interceptor != nil {
		serverOptions = append(serverOptions, grpc.StreamInterceptor(s.interceptor))
	}

	if s.destinationCreds != nil {
		serverOptions = append(serverOptions, grpc.Creds(s.destinationCreds))
	}

	s.grpcServer = grpc.NewServer(serverOptions...)

	return s, nil
}

func (s *server) Serve(listener net.Listener) error {
	wrappedProxy := grpcweb.WrapServer(
		s.grpcServer,
		grpcweb.WithCorsForRegisteredEndpointsOnly(false), // because we are proxying
	)

	httpServer := http.Server{
		Handler: http.HandlerFunc(wrappedProxy.ServeHTTP),
	}

	if s.certFile != "" && s.keyFile != "" {
		return httpServer.ServeTLS(listener, s.certFile, s.keyFile)
	}

	return httpServer.Serve(listener)
}
