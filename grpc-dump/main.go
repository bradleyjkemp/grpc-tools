package main

import (
	"flag"
	"fmt"
	"github.com/bradleyjkemp/grpc-tools/grpc-proxy"
	"github.com/bradleyjkemp/grpc-tools/internal/proto_descriptor"
	_ "github.com/bradleyjkemp/grpc-tools/internal/versionflag"
	"github.com/jhump/protoreflect/desc"
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
	var knownMethods map[string]*desc.MethodDescriptor
	if *protoRoots != "" {
		descs, err := proto_descriptor.LoadProtoDirectories(strings.Split(*protoRoots, ",")...)
		if err != nil {
			return err
		} else {
			fmt.Fprintln(os.Stderr, "Loaded", len(descs), "service descriptors")
			knownMethods = descs
		}
	}
	if *protoDescriptors != "" {
		descs, err := proto_descriptor.LoadProtoDescriptors(strings.Split(*protoDescriptors, ",")...)
		if err != nil {
			return err
		} else {
			fmt.Fprintln(os.Stderr, "Loaded", len(descs), "service descriptors")
			knownMethods = descs
		}
	}

	proxy, err := grpc_proxy.New(
		grpc_proxy.WithInterceptor(dumpInterceptor(knownMethods)),
		grpc_proxy.DefaultFlags(),
	)
	if err != nil {
		return err
	}

	return proxy.Start()
}
