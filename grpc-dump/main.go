package main

import (
	"flag"
	"fmt"
	"github.com/bradleyjkemp/grpc-tools/grpc-proxy"
	"github.com/bradleyjkemp/grpc-tools/internal"
	"github.com/jhump/protoreflect/desc"
	"net"
	"os"
	"strings"
)

var (
	port              = flag.Int("port", 0, "Port to listen on")
	certFile          = flag.String("cert", "", "Certificate file to use for serving using TLS")
	keyFile           = flag.String("key", "", "Key file to use for serving using TLS")
	destinationServer = flag.String("destination", "", "Destination server to forward requests to")
	protoRoots        = flag.String("proto_roots", "", "A comma separated list of directories to search for gRPC service definitions")
)

func main() {
	flag.Parse()
	err := run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		flag.Usage()
		os.Exit(1)
	}
}

//[]string{
//"/Users/bradleykemp/repos/platform/proto",
//"/Users/bradleykemp/go/src/github.com/googleapis/googleapis",
//"/Users/bradleykemp/go/src/",
//}...

func run() error {
	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", *port))
	if err != nil {
		return err
	}

	var knownMethods map[string]*desc.MethodDescriptor
	if *protoRoots != "" {
		descs, err := internal.LoadServiceDescriptors(strings.Split(*protoRoots, ",")...)
		if err != nil {
			return err
		} else {
			fmt.Fprintln(os.Stderr, "Loaded", len(descs), "service descriptors")
			knownMethods = descs
		}
	}

	options := []grpc_proxy.Configurator{
		grpc_proxy.WithInterceptor(dumpInterceptor(knownMethods)),
	}

	if *certFile != "" || *keyFile != "" {
		if *certFile == "" || *keyFile == "" {
			return fmt.Errorf("both or neither of --cert and --key must be specified")
		}

		options = append(options, grpc_proxy.UsingTLS(*certFile, *keyFile))
	}

	if *destinationServer != "" {
		options = append(options, grpc_proxy.DestinationServer(*destinationServer))
	}

	proxy, err := grpc_proxy.New(options...)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Listening on %s\n", lis.Addr())
	return proxy.Serve(lis)
}
