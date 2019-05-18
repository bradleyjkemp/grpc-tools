package grpc_proxy

import (
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
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
	listener         net.Listener
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
	s.listener = listener

	wrappedProxy := grpcweb.WrapServer(
		s.grpcServer,
		grpcweb.WithCorsForRegisteredEndpointsOnly(false), // because we are proxying
	)
	httpServer := s.newHttpServer(wrappedProxy, false)
	httpsServer := s.newHttpServer(wrappedProxy, true)

	httpLis, httpsLis := newHttpHttpsMux(listener)

	if s.certFile != "" && s.keyFile != "" {
		go httpsServer.ServeTLS(httpsLis, s.certFile, s.keyFile)
	}

	// Unencrypted HTTP2 is not supported by default so need this wrapper
	// This accepts PRI methods and does the necessary upgrade
	httpServer.Handler = h2c.NewHandler(httpServer.Handler, &http2.Server{})
	return httpServer.Serve(httpLis)
}

func (s *server) handleConnect(w http.ResponseWriter, r *http.Request) {
	destConn, err := net.DialTimeout(s.listener.Addr().Network(), s.listener.Addr().String(), 10*time.Second)
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
}

func (s *server) newHttpServer(wrappedProxy *grpcweb.WrappedGrpcServer, listensOnSSL bool) *http.Server {
	httpReverseProxy := &httputil.ReverseProxy{
		Director: func(request *http.Request) {
			if listensOnSSL {
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
			r.Context()
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
