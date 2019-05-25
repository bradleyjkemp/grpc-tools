package main

import (
	"flag"
	"fmt"
	"github.com/bradleyjkemp/grpc-tools/grpc-proxy"
	"os"
)

var (
	dumpPath = flag.String("dump", "", "gRPC dump to serve requests from")
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
	interceptor, err := loadFixture(*dumpPath)
	if err != nil {
		return err
	}

	proxy, err := grpc_proxy.New(
		grpc_proxy.WithInterceptor(interceptor.intercept),
		grpc_proxy.DefaultFlags(),
	)
	if err != nil {
		return err
	}

	return proxy.Start()
}
