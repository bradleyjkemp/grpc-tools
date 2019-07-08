package main

import (
	"fmt"
	"github.com/bradleyjkemp/grpc-tools/grpc-dump/dump"
	"github.com/bradleyjkemp/grpc-tools/grpc-fixture/fixture"
	"github.com/bradleyjkemp/grpc-tools/grpc-proxy"
	"github.com/bradleyjkemp/grpc-tools/grpc-replay/replay"
	"github.com/bradleyjkemp/grpc-tools/internal/proxydialer"
	"net/url"
	"os/exec"
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

	cmd := curlCommand("http://grpc-web.github.io/grpc.gateway.testing.EchoService/Echo")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatal("Unexpected error:", err, string(out))
	}

	cmd = curlCommand("https://grpc-web.github.io:1234/grpc.gateway.testing.EchoService/Echo")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatal("Unexpected error:", err, string(out))
	}
}

func curlCommand(url string) *exec.Cmd {
	cmd := exec.Command("curl", "-X", "POST", url, "-H", "Pragma: no-cache", "-H", "X-User-Agent: grpc-web-javascript/0.1", "-H", "Origin: http://localhost:8081", "-H", "Accept-Encoding: gzip, deflate, br", "-H", "Accept-Language: en-US,en;q=0.9", "-H", "custom-header-1: value1", "-H", "User-Agent: Mozilla/5.0", "-H", "Content-Type: application/grpc-web+proto", "-H", "Accept: */*", "-H", "X-Grpc-Web: 1", "-H", "Cache-Control: no-cache", "-H", "Referer: http://localhost:8081/echotest.html", "-H", "Connection: keep-alive")
	cmd.Env = []string{fmt.Sprintf("all_proxy=http://localhost:%d", dumpPort)}
	return cmd
}
