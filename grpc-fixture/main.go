package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/bradleyjkemp/grpc-tools/grpc-proxy"
	"github.com/bradleyjkemp/grpc-tools/pkg"
	"io"
	"net"
	"os"
)

var (
	port     = flag.Int("port", 0, "Port to listen on")
	certFile = flag.String("cert", "", "Certificate file to use for serving using TLS")
	keyFile  = flag.String("key", "", "Key file to use for serving using TLS")
	dumpPath = flag.String("dump", "", "gRPC dump to serve requests from")
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
	interceptor, err := loadFixture()
	if err != nil {
		return err
	}

	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", *port))
	if err != nil {
		return err
	}
	if *port == 0 {
		// port was auto-selected so need to tell the user
		fmt.Fprintf(os.Stderr, "listening on %s\n", lis.Addr())
	}

	options := []grpc_proxy.Configurator{
		grpc_proxy.WithInterceptor(interceptor.intercept),
	}

	if *certFile != "" || *keyFile != "" {
		if *certFile == "" || *keyFile == "" {
			return fmt.Errorf("both or neither of --cert and --key must be specified")
		}
		options = append(options, grpc_proxy.UsingTLS(*certFile, *keyFile))
	}

	proxy, err := grpc_proxy.New(options...)
	if err != nil {
		return err
	}

	return proxy.Serve(lis)
}

func loadFixture() (*fixtureInterceptor, error) {
	dumpFile, err := os.Open(*dumpPath)
	if err != nil {
		return nil, err
	}

	dumpDecoder := json.NewDecoder(dumpFile)
	interceptor := &fixtureInterceptor{
		allRecordedMethods: map[string][][]pkg.StreamEvent{},
		unaryMethods:       map[string]map[string]string{},
	}
	rpc := pkg.RPC{}
	for {
		err := dumpDecoder.Decode(&rpc)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		key := rpc.Service + "/" + rpc.Method
		interceptor.allRecordedMethods[key] = append(interceptor.allRecordedMethods[key], rpc.Messages)
	}

	for method, calls := range interceptor.allRecordedMethods {
		isUnary := true
		for _, messages := range calls {
			// exactly two messages: client request and server response
			isUnary = isUnary &&
				(len(messages) == 2 &&
					messages[0].ClientMessage != nil && messages[0].ServerMessage == nil &&
					messages[1].ClientMessage == nil && messages[1].ServerMessage != nil)
		}
		if isUnary {
			// all requests looked unary so can add a shortcut
			for _, request := range calls {
				interceptor.unaryMethods[method][string(request[0].ClientMessage)] = string(request[1].ServerMessage)
			}
		}
	}

	return interceptor, nil
}
