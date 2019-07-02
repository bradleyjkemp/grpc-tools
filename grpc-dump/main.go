package main

import (
	"flag"
	"fmt"
	"github.com/bradleyjkemp/grpc-tools/grpc-proxy"
	"github.com/bradleyjkemp/grpc-tools/internal/proto_decoder"
	_ "github.com/bradleyjkemp/grpc-tools/internal/versionflag"
	"os"
	"strings"
)

var (
	protoRoots       = flag.String("proto_roots", "", "A comma separated list of directories to search for gRPC service definitions.")
	protoDescriptors = flag.String("proto_descriptors", "", "A comma separated list of proto descriptors to load gRPC service definitions from.")
)

func main() {
	grpc_proxy.RegisterDefaultFlags()
	flag.Parse()
	err := run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		flag.Usage()
		os.Exit(1)
	}
}

func run() error {
	var resolvers []proto_decoder.MessageResolver
	if *protoRoots != "" {
		r, err := proto_decoder.NewFileResolver(strings.Split(*protoRoots, ",")...)
		if err != nil {
			return err
		}
		resolvers = append(resolvers, r)
	}
	if *protoDescriptors != "" {
		r, err := proto_decoder.NewDescriptorResolver(strings.Split(*protoRoots, ",")...)
		if err != nil {
			return err
		}
		resolvers = append(resolvers, r)
	}
	// Always use the unknown message resolver
	resolvers = append(resolvers, proto_decoder.NewUnknownResolver())

	proxy, err := grpc_proxy.New(
		grpc_proxy.WithInterceptor(dumpInterceptor(proto_decoder.NewDecoder(resolvers...))),
		grpc_proxy.DefaultFlags(),
	)
	if err != nil {
		return err
	}

	return proxy.Start()
}
