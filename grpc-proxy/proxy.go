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
	fmt.Fprintf(os.Stderr, "Listening on %s\n", listener.Addr()) // TODO move this to a logger controllable by options
	proxyLis := &proxyListener{
		channel:     s.proxiedConns,
		TCPListener: listener.(*net.TCPListener),
	}

	grpcWebHandler := grpcweb.WrapServer(
		grpc.NewServer(s.serverOptions...),
		grpcweb.WithCorsForRegisteredEndpointsOnly(false), // because we are proxying
	)
	httpServer := s.newHttpServer(grpcWebHandler)
	httpsServer := withHttpsMarkerMiddleware(s.newHttpServer(grpcWebHandler))

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

	httpLis, httpsLis := newTlsMux(proxyLis, x509Cert)
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
	var conn tlsMuxConn
	if conn, ok = clientConn.(tlsMuxConn); !ok {
		fmt.Fprintf(os.Stderr, "Err: unknown connection type: %v\n", clientConn)
		clientConn.Close()
		return
	}

	s.proxiedConns <- &proxiedConn{conn, r.Host, conn.tls}
}

var httpReverseProxy = &httputil.ReverseProxy{
	Director: func(request *http.Request) {
		// Because of the TLSmux used to server HTTP and HTTPS on the same port
		// we have to rely on the Forwarded header (added by middleware) to
		// tell which protocol to use for proxying.
		// (we could always set HTTP but would mean relying on the upstream
		// properly redirecting HTTP->HTTPS)
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

func isGrpcRequest(server *grpcweb.WrappedGrpcServer, r *http.Request) bool {
	return server.IsAcceptableGrpcCorsRequest(r) || // CORS request from browser
		server.IsGrpcWebRequest(r) || // gRPC-Web request from browser
		r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") // Standard gRPC request
}

func (s *server) newHttpServer(grpcHandler *grpcweb.WrappedGrpcServer) *http.Server {
	return &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodConnect:
				s.handleConnect(w, r)
			case isGrpcRequest(grpcHandler, r):
				grpcHandler.ServeHTTP(w, r)
			default:
				// Many clients use a mix of gRPC and non-gRPC requests
				// so must try to be as transparent as possible for normal
				// HTTP requests by proxying the request to the original destination.
				httpReverseProxy.ServeHTTP(w, r)
			}
		}),
	}
}

func withHttpsMarkerMiddleware(server *http.Server) *http.Server {
	wrappedHandler := server.Handler
	server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tls.AddHTTPSMarker(r.Header)
		wrappedHandler.ServeHTTP(w, r)
	})
	return server
}
