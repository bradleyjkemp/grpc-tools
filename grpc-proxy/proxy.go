package grpc_proxy

import (
	crypto_tls "crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/bradleyjkemp/grpc-tools/internal"
	"github.com/bradleyjkemp/grpc-tools/internal/tls"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
)

var proxyStreamDesc = &grpc.StreamDesc{
	ServerStreams: true,
	ClientStreams: true,
}

type server struct {
	serverOptions []grpc.ServerOption

	grpcServer   *grpc.Server
	proxiedConns chan *proxiedConn // TODO: properly scope this to only where it's used

	port     int
	certFile string
	keyFile  string

	destination string
	connPool    *internal.ConnPool

	listener net.Listener
}

func New(configurators ...Configurator) (*server, error) {
	s := &server{
		proxiedConns: make(chan *proxiedConn),
		connPool:     internal.NewConnPool(),
	}
	s.serverOptions = []grpc.ServerOption{
		grpc.CustomCodec(NoopCodec{}),              // Allows for passing raw []byte messages around
		grpc.UnknownServiceHandler(s.proxyHandler), // All services are unknown so will be proxied
	}

	for _, configurator := range configurators {
		configurator(s)
	}

	return s, nil
}

func (s *server) Start() error {
	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", s.port))
	if err != nil {
		return fmt.Errorf("failed to listen on port (%d): %v", s.port, err)
	}
	if _, ok := listener.(*net.TCPListener); !ok {
		return fmt.Errorf("can only listen on a TCP listener")
	}
	fmt.Fprintf(os.Stderr, "Listening on %s\n", listener.Addr()) // TODO move this to a logger controllable by options
	proxyLis := &proxyListener{
		channel:     s.proxiedConns,
		TCPListener: listener.(*net.TCPListener),
	}

	wrappedProxy := grpcweb.WrapServer(
		grpc.NewServer(s.serverOptions...),
		grpcweb.WithCorsForRegisteredEndpointsOnly(false), // because we are proxying
	)
	httpServer := s.newHttpServer(wrappedProxy, false)
	httpsServer := s.newHttpServer(wrappedProxy, true)

	var x509Cert *x509.Certificate
	if s.certFile != "" && s.keyFile != "" {
		// TODO: load this as part of New()
		tlsCert, err := crypto_tls.LoadX509KeyPair(s.certFile, s.keyFile)
		if err != nil {
			return err
		}

		x509Cert, err = x509.ParseCertificate(tlsCert.Certificate[0]) //TODO do we need to parse anything other than [0]?
		if err != nil {
			return err
		}
	}

	httpLis, httpsLis := newHttpHttpsMux(proxyLis, x509Cert)
	go httpsServer.ServeTLS(httpsLis, s.certFile, s.keyFile)

	// Unencrypted HTTP2 is not supported by default so need this wrapper
	// This accepts PRI methods and does the necessary upgrade
	httpServer.Handler = h2c.NewHandler(httpServer.Handler, &http2.Server{})
	return httpServer.Serve(httpLis)
}

func (s *server) handleConnect(w http.ResponseWriter, r *http.Request) {
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
	var bidiConn bidirectionalConn
	switch conn := clientConn.(type) {
	case *net.TCPConn:
		bidiConn = conn
	case cmuxConn:
		bidiConn = conn
	default:
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	s.proxiedConns <- &proxiedConn{bidiConn, r.Host}
}

func (s *server) newHttpServer(wrappedProxy *grpcweb.WrappedGrpcServer, listensOnTLS bool) *http.Server {
	httpReverseProxy := &httputil.ReverseProxy{
		Director: func(request *http.Request) {
			if tls.IsTLSRequest(request.Header) {
				request.URL.Scheme = "https"
			} else {
				request.URL.Scheme = "http"
			}
			request.URL.Host = request.Host
		},
		ModifyResponse: func(response *http.Response) error {
			return nil
		},
	}
	return &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if listensOnTLS {
				tls.AddHTTPSMarker(r.Header)
			}
			switch {
			case r.Method == http.MethodConnect:
				s.handleConnect(w, r)
			case wrappedProxy.IsGrpcWebRequest(r) || r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc"):
				wrappedProxy.ServeHTTP(w, r)
			default:
				httpReverseProxy.ServeHTTP(w, r)
			}
		}),
	}
}
