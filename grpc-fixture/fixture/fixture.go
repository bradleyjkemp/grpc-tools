package fixture

import (
	"github.com/bradleyjkemp/grpc-tools/grpc-proxy"
	"github.com/bradleyjkemp/grpc-tools/internal/proto_decoder"
	"strings"
)

// Run is exported for testing
func Run(protoRoots, protoDescriptors, dumpPath string, proxyConfig ...grpc_proxy.Configurator) error {
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
	encoder := proto_decoder.NewEncoder(resolvers...)

	interceptor, err := loadFixture(dumpPath, encoder)
	if err != nil {
		return err
	}

	proxy, err := grpc_proxy.New(
		append(proxyConfig, grpc_proxy.WithInterceptor(interceptor.intercept))...,
	)
	if err != nil {
		return err
	}

	return proxy.Start()
}
