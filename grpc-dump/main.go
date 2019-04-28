package main

import (
	"flag"
	"fmt"
	"github.com/bradleyjkemp/grpc-tools/grpc-proxy"
	"net"
	"os"
)

var (
	port              = flag.Int("port", 0, "Port to listen on")
	certFile          = flag.String("cert", "", "Certificate file to use for serving using TLS")
	keyFile           = flag.String("key", "", "Key file to use for serving using TLS")
	destinationServer = flag.String("destination", "", "Destination server to forward requests to")
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

func run() error {
	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", *port))
	if err != nil {
		return err
	}
	if *port == 0 {
		// port was auto-selected so need to tell the user
		fmt.Fprintf(os.Stderr, "listening on %s\n", lis.Addr())
	}

	options := []grpc_proxy.Configurator{
		grpc_proxy.WithInterceptor(dumpInterceptor),
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

	return proxy.Serve(lis)
}
