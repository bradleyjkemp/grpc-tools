package grpc_proxy

import (
	crypto_tls "crypto/tls"
	"crypto/x509"
	"github.com/bradleyjkemp/grpc-tools/internal"
	"github.com/bradleyjkemp/grpc-tools/internal/tls"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
)

var proxyStreamDesc = &grpc.StreamDesc{
	ServerStreams: true,
	ClientStreams: true,
}

type server struct {
	err           error // filled if any errors occur during startup
	serverOptions []grpc.ServerOption

	grpcServer   *grpc.Server
	httpServer   *http.Server
	proxiedConns chan *proxiedConn
	certFile     string
	keyFile      string

	destination *grpc.ClientConn
	connPool    *internal.ConnPool

	interceptor grpc.StreamServerInterceptor
}

func New(configurators ...Configurator) (*server, error) {
	s := &server{
		proxiedConns: make(chan *proxiedConn),
		connPool:     internal.NewConnPool(),
	}
	for _, configurator := range configurators {
		configurator(s)
	}

	if s.err != nil {
		return nil, s.err
	}

	serverOptions := append(s.serverOptions,
		grpc.CustomCodec(NoopCodec{}),              // Allows for passing raw []byte messages around
		grpc.UnknownServiceHandler(s.proxyHandler), // All services are unknown so will be proxied
	)

	if s.interceptor != nil {
		serverOptions = append(serverOptions, grpc.StreamInterceptor(s.interceptor))
	}

	s.grpcServer = grpc.NewServer(serverOptions...)

	return s, nil
}

func (s *server) Serve(listener net.Listener) error {
	wrappedProxy := grpcweb.WrapServer(
		s.grpcServer,
		grpcweb.WithCorsForRegisteredEndpointsOnly(false), // because we are proxying
	)
	httpServer := s.newHttpServer(wrappedProxy, false)
	httpsServer := s.newHttpServer(wrappedProxy, true)

	var x509Cert *x509.Certificate
	if s.certFile != "" && s.keyFile != "" {
		tlsCert, err := crypto_tls.LoadX509KeyPair(s.certFile, s.keyFile)
		if err != nil {
			return err
		}
		httpsServer.TLSConfig = &crypto_tls.Config{
			Certificates: []crypto_tls.Certificate{
				tlsCert,
			},
		}
		x509Cert, err = x509.ParseCertificate(tlsCert.Certificate[0]) //TODO do we need to parse anything other than [0]?
		if err != nil {
			return err
		}
	}

	httpLis, httpsLis := newHttpHttpsMux(&proxyListener{
		channel:  s.proxiedConns,
		Listener: listener,
	}, x509Cert)

	if s.certFile != "" && s.keyFile != "" {
		// already loaded and configured certs above so no need to specify again
		go httpsServer.ServeTLS(httpsLis, "", "")
	}

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
	s.proxiedConns <- &proxiedConn{clientConn, r.Host}
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
