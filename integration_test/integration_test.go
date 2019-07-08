package main

import (
	"fmt"
	"github.com/bradleyjkemp/grpc-tools/grpc-dump/dump"
	"github.com/bradleyjkemp/grpc-tools/grpc-fixture/fixture"
	"github.com/bradleyjkemp/grpc-tools/grpc-proxy"
	"github.com/bradleyjkemp/grpc-tools/grpc-replay/replay"
	"github.com/bradleyjkemp/grpc-tools/internal/proxydialer"
	"net/url"
	"testing"
)

const (
	protoRoots       = "."
	protoDescriptors = ""
	certFile         = "_wildcard.github.io.pem"
	keyFile          = "_wildcard.github.io-key.pem"

	fixturePort = 16353
	dumpPort    = 16354
)

func TestIntegration(t *testing.T) {
	go func() {
		fixtureErr := fixture.Run(
			protoRoots,
			protoDescriptors,
			"test-golden.json",
			grpc_proxy.Port(fixturePort),
			grpc_proxy.UsingTLS(certFile, keyFile),
		)
		if fixtureErr != nil {
			t.Fatal("Unexpected error:", fixtureErr)
		}
	}()

	// TODO: inject http_proxy settings here
	go func() {
		dumpErr := dump.Run(
			protoRoots,
			protoDescriptors,
			grpc_proxy.Port(dumpPort),
			grpc_proxy.UsingTLS(certFile, keyFile),
			grpc_proxy.WithDialer(proxydialer.NewProxyDialer(func(req *url.URL) (*url.URL, error) {
				return &url.URL{
					Host: fmt.Sprintf("localhost:%d", fixturePort),
				}, nil
			})),
		)
		if dumpErr != nil {
			t.Fatal("Unexpected error:", dumpErr)
		}
	}()

	replayErr := replay.Run(
		protoRoots,
		protoDescriptors,
		"test-dump.json",
		"",
		proxydialer.NewProxyDialer(func(req *url.URL) (*url.URL, error) {
			return &url.URL{
				Host: fmt.Sprintf("localhost:%d", dumpPort),
			}, nil
		}),
	)
	if replayErr != nil {
		t.Fatal("Unexpected error:", replayErr)
	}
}
