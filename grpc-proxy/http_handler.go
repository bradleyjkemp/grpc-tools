package grpc_proxy

import (
	"github.com/bradleyjkemp/grpc-tools/internal/marker"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
)

func newHttpServer(logger *logrus.Logger, grpcHandler *grpcweb.WrappedGrpcServer, internalRedirect func(net.Conn, string)) *http.Server {
	return &http.Server{
		Handler: h2c.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodConnect:
				logger.Debug("Handling HTTP CONNECT request for destination ", r.URL)
				handleConnect(w, r, internalRedirect)
			case isGrpcRequest(grpcHandler, r):
				logger.Debug("Handling gRPC request ", r.URL)
				grpcHandler.ServeHTTP(w, r)
			default:
				// Many clients use a mix of gRPC and non-gRPC requests
				// so must try to be as transparent as possible for normal
				// HTTP requests by proxying the request to the original destination.
				logger.Debugf("Reverse proxying HTTP request %s %s %s", r.Method, r.Host, r.URL)
				httpReverseProxy.ServeHTTP(w, r)
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
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	clientConn.Write([]byte("HTTP/1.0 200 OK\r\n\r\n"))
	internalRedirect(clientConn, r.Host)
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

func isGrpcRequest(server *grpcweb.WrappedGrpcServer, r *http.Request) bool {
	return server.IsAcceptableGrpcCorsRequest(r) || // CORS request from browser
		server.IsGrpcWebRequest(r) || // gRPC-Web request from browser
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
