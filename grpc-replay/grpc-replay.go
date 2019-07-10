package main

import (
	"flag"
	"fmt"
	"github.com/bradleyjkemp/grpc-tools/grpc-replay/replay"
	"github.com/bradleyjkemp/grpc-tools/internal/proxydialer"
	_ "github.com/bradleyjkemp/grpc-tools/internal/versionflag"
	"golang.org/x/net/http/httpproxy"
	"os"
)

func main() {
	var (
		destinationOverride = flag.String("destination", "", "Destination server to forward requests to. By default the destination for each RPC is autodetected from the dump metadata.")
		dumpPath            = flag.String("dump", "", "The gRPC dump to replay requests from")
		protoRoots          = flag.String("proto_roots", "", "A comma separated list of directories to search for gRPC service definitions.")
		protoDescriptors    = flag.String("proto_descriptors", "", "A comma separated list of proto descriptors to load gRPC service definitions from.")
	)

	flag.Parse()
	err := replay.Run(*protoRoots, *protoDescriptors, *dumpPath, *destinationOverride, proxydialer.NewProxyDialer(httpproxy.FromEnvironment().ProxyFunc()))
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		flag.Usage()
		os.Exit(1)
	}
}
