package dump

import (
	"github.com/bradleyjkemp/grpc-tools/grpc-proxy"
	"github.com/bradleyjkemp/grpc-tools/internal/proto_decoder"
	"strings"
)

func Run(protoRoots, protoDescriptors string, proxyConfig ...grpc_proxy.Configurator) error {
	var resolvers []proto_decoder.MessageResolver
	if protoRoots != "" {
		r, err := proto_decoder.NewFileResolver(strings.Split(protoRoots, ",")...)
		if err != nil {
			return err
		}
		resolvers = append(resolvers, r)
	}
	if protoDescriptors != "" {
		r, err := proto_decoder.NewDescriptorResolver(strings.Split(protoRoots, ",")...)
		if err != nil {
			return err
		}
		resolvers = append(resolvers, r)
	}
	// Always use the unknown message resolver
	resolvers = append(resolvers, proto_decoder.NewUnknownResolver())

	opts := append(proxyConfig, grpc_proxy.WithInterceptor(dumpInterceptor(proto_decoder.NewDecoder(resolvers...))))
	proxy, err := grpc_proxy.New(
		opts...,
	)
	if err != nil {
		return err
	}

	return proxy.Start()
}