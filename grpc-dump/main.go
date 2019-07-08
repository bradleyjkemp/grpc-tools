package main

import (
	"flag"
	"fmt"
	"github.com/bradleyjkemp/grpc-tools/grpc-dump/dump"
	"github.com/bradleyjkemp/grpc-tools/grpc-proxy"
	_ "github.com/bradleyjkemp/grpc-tools/internal/versionflag"
	"os"
)

var (
	protoRoots       = flag.String("proto_roots", "", "A comma separated list of directories to search for gRPC service definitions.")
	protoDescriptors = flag.String("proto_descriptors", "", "A comma separated list of proto descriptors to load gRPC service definitions from.")
)

func main() {
	grpc_proxy.RegisterDefaultFlags()
	flag.Parse()
	err := dump.Run(*protoRoots, *protoDescriptors, grpc_proxy.DefaultFlags())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		flag.Usage()
		os.Exit(1)
	}
}
