package main

import (
	"flag"
	"fmt"
	"github.com/bradleyjkemp/grpc-tools/grpc-fixture/fixture"
	"github.com/bradleyjkemp/grpc-tools/grpc-proxy"
	_ "github.com/bradleyjkemp/grpc-tools/internal/versionflag"
	"os"
)

func main() {
	var (
		dumpPath         = flag.String("dump", "", "gRPC dump to serve requests from")
		protoRoots       = flag.String("proto_roots", "", "A comma separated list of directories to search for gRPC service definitions.")
		protoDescriptors = flag.String("proto_descriptors", "", "A comma separated list of proto descriptors to load gRPC service definitions from.")
	)

	grpc_proxy.RegisterDefaultFlags()
	flag.Parse()
	err := fixture.Run(*protoRoots, *protoDescriptors, *dumpPath, grpc_proxy.DefaultFlags())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		flag.Usage()
		os.Exit(1)
	}
}
