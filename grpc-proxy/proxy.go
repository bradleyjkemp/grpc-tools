package grpc_proxy

import (
	"fmt"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"io"
	"net"
	"net/http"
	"time"
)

var proxyStreamDesc = &grpc.StreamDesc{
	ServerStreams: true,
	ClientStreams: true,
}

type readWriteCloser interface {
	io.ReadWriter
	CloseRead() error
	CloseWrite() error
}

type server struct {
	err              error // filled if any errors occur during startup
	destinationCreds credentials.TransportCredentials
	serverOptions    []grpc.ServerOption
	grpcServer       *grpc.Server
	httpServer       *http.Server
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

	httpServer := &http.Server{}

	httpServer.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("got req", r.Method, r.Host, r.URL)
		if r.Method == http.MethodConnect {
			destConn, err := net.DialTimeout(listener.Addr().Network(), listener.Addr().String(), 10*time.Second)
			if err != nil {
				http.Error(w, err.Error(), http.StatusServiceUnavailable)
				return
			}

			hijacker, ok := w.(http.Hijacker)
			if !ok {
				http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
				return
			}
			clientConn, _, err := hijacker.Hijack()
			if err != nil {
				http.Error(w, err.Error(), http.StatusServiceUnavailable)
				return
			}
			clientConn.Write([]byte("HTTP/1.0 200 OK\r\n\r\n"))

			clientTCP := clientConn.(readWriteCloser)
			destTCP := destConn.(readWriteCloser)
			go func() {
				io.Copy(destTCP, clientTCP)
				destTCP.CloseWrite()
				clientTCP.CloseRead()
			}()
			go func() {
				io.Copy(clientTCP, destTCP)
				clientTCP.CloseWrite()
				destTCP.CloseRead()
			}()
		} else {
			wrappedProxy.ServeHTTP(w, r)
		}
	})

	httpLis, httpsLis := newHttpHttpsMux(listener)

	if s.certFile != "" && s.keyFile != "" {
		go httpServer.ServeTLS(httpsLis, s.certFile, s.keyFile)
	}

	// Unencrypted HTTP2 is not supported by default so need this wrapper
	// This accepts PRI methods and does the necessary upgrade
	httpServer.Handler = h2c.NewHandler(httpServer.Handler, &http2.Server{})
	return httpServer.Serve(httpLis)
}
