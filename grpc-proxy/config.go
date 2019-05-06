package grpc_proxy

import (
	"context"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type Configurator func(*server)

func DestinationServer(target string, dialOptions ...grpc.DialOption) Configurator {
	return func(s *server) {
		ctx := context.TODO()
		dialOptions = append(
			dialOptions,
			grpc.WithTransportCredentials(credentials.NewTLS(nil)),
			grpc.WithDefaultCallOptions(grpc.ForceCodec(NoopCodec{})),
		)
		var err error
		s.destination, err = grpc.DialContext(ctx, target, dialOptions...)

		if err != nil {
			s.err = errors.Wrap(err, "failed to dial destination server")
		}
	}
}

func WithOptions(options ...grpc.ServerOption) Configurator {
	return func(s *server) {
		s.serverOptions = append(s.serverOptions, options...)
	}
}

func WithInterceptor(interceptor grpc.StreamServerInterceptor) Configurator {
	return func(s *server) {
		s.interceptor = interceptor
	}
}

func UsingTLS(certFile, keyFile string) Configurator {
	return func(s *server) {
		s.certFile = certFile
		s.keyFile = keyFile
	}
}
