package grpc_proxy

import (
	"flag"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type Configurator func(*server)

func WithOptions(options ...grpc.ServerOption) Configurator {
	return func(s *server) {
		s.serverOptions = append(s.serverOptions, options...)
	}
}

func WithInterceptor(interceptor grpc.StreamServerInterceptor) Configurator {
	return func(s *server) {
		s.serverOptions = append(s.serverOptions, grpc.StreamInterceptor(interceptor))
	}
}

func UsingTLS(certFile, keyFile string) Configurator {
	return func(s *server) {
		s.certFile = certFile
		s.keyFile = keyFile
	}
}

func Port(port int) Configurator {
	return func(s *server) {
		s.port = port
	}
}

func WithDialer(dialer ContextDialer) Configurator {
	return func(s *server) {
		s.dialer = dialer
	}
}

var (
	fPort        int
	fCertFile    string
	fKeyFile     string
	fDestination string
	fLogLevel    string
)

// Must be called before flag.Parse() if using the DefaultFlags option
func RegisterDefaultFlags() {
	flag.IntVar(&fPort, "port", 0, "Port to listen on.")
	flag.StringVar(&fCertFile, "cert", "", "Certificate file to use for serving using TLS. By default the current directory will be scanned for mkcert certificates to use.")
	flag.StringVar(&fKeyFile, "key", "", "Key file to use for serving using TLS. By default the current directory will be scanned for mkcert keys to use.")
	flag.StringVar(&fDestination, "destination", "", "Destination server to forward requests to if no destination can be inferred from the request itself. This is generally only used for clients not supporting HTTP proxies.")
	flag.StringVar(&fLogLevel, "log_level", logrus.InfoLevel.String(), "Set the log level that grpc-proxy will log at. Values are {error, warning, info, debug}")
}

// This must be used after a call to flag.Parse()
func DefaultFlags() Configurator {
	return func(s *server) {
		s.port = fPort
		s.certFile = fCertFile
		s.keyFile = fKeyFile
		s.destination = fDestination
	}
}
