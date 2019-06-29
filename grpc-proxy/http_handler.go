package grpc_proxy

import (
	"fmt"
	"github.com/bradleyjkemp/grpc-tools/internal/marker"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
)

type grpcWebServer interface {
	ServeHTTP(resp http.ResponseWriter, req *http.Request)
	IsGrpcWebRequest(req *http.Request) bool
}

func newHttpServer(logger logrus.FieldLogger, grpcHandler grpcWebServer, internalRedirect func(net.Conn, string), reverseProxy http.Handler) *http.Server {
	return &http.Server{
		Handler: h2c.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodConnect:
				logger.Debug("Handling HTTP CONNECT request for destination ", r.URL)
				handleConnect(w, r, internalRedirect)
			case isGrpcRequest(grpcHandler, r):
				logger.Debug("Handling gRPC request ", r.URL)
				// This request may be a gRPC-Web request that came in on HTTP/1.X
				// So delete any legacy headers that will cause gRPC to break
				// I don't really know why this works but without these lines the integration tests fail
				// with error:
				// Bad Request: HTTP status code 400; transport: received the unexpected content-type \"text/plain; charset=utf-8\"
				r.Header.Del("Connection")
				r.Header.Del("Proxy-Connection")
				grpcHandler.ServeHTTP(w, r)
			default:
				// Many clients use a mix of gRPC and non-gRPC requests
				// so must try to be as transparent as possible for normal
				// HTTP requests by proxying the request to the original destination.
				logger.Debugf("Reverse proxying HTTP request %s %s %s", r.Method, r.Host, r.URL)
				reverseProxy.ServeHTTP(w, r)
			}
		}), &http2.Server{}),
	}
}

func handleConnect(w http.ResponseWriter, r *http.Request, internalRedirect func(net.Conn, string)) {
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}
	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		// TODO: log error here
		return
	}
	_, err = fmt.Fprintf(clientConn, "%s 200 OK\r\n\r\n", r.Proto)
	if err == nil {
		internalRedirect(clientConn, r.Host)
	} else {
		_ = clientConn.Close()
	}
}

var httpReverseProxy = &httputil.ReverseProxy{
	Director: func(request *http.Request) {
		// Because of the TLSmux used to server HTTP and HTTPS on the same port
		// we have to rely on the Forwarded header (added by middleware) to
		// tell which protocol to use for proxying.
		// (we could always set HTTP but would mean relying on the upstream
		// properly redirecting HTTP->HTTPS)
		if marker.IsTLSRequest(request.Header) {
			request.URL.Scheme = "https"
		} else {
			request.URL.Scheme = "http"
		}
		request.URL.Host = request.Host
	},
}

func isGrpcRequest(server grpcWebServer, r *http.Request) bool {
	return server.IsGrpcWebRequest(r) || // gRPC-Web request from browser
		r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") // Standard gRPC request
}

// Knowing whether a request came in over HTTP or HTTPS
// is important for being able to replay the request.
// This adds a Forwarded header with the protocol information.
//
// It also adds a wrapper to enable HTTP2 on "unencrypted" connections.
// (not actually unencrypted because we're using a TLS listener)
func withHttpsMiddleware(server *http.Server) *http.Server {
	wrappedHandler := server.Handler
	server.Handler = h2c.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		marker.AddHTTPSMarker(r.Header)
		wrappedHandler.ServeHTTP(w, r)
	}), &http2.Server{})

	return server
}
