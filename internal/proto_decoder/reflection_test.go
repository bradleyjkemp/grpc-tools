package proto_decoder

import (
	"context"
	"github.com/bradleyjkemp/grpc-tools/internal"
	"github.com/bradleyjkemp/grpc-tools/internal/test_server"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/metadata"
	"net"
	"testing"
	"time"
)

func TestReflectionResolver(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		if err := test_server.Listen(ctx, lis); err != nil {
			t.Fatal(err)
		}
	}()

	r := NewReflectionResolver(logrus.New())
	descriptor, err := r.resolveEncoded(ctx, "/test_protos.TestService/TestUnaryClientRequest", &internal.Message{
		MessageOrigin: internal.ClientMessage,
	}, metadata.Pairs(":authority", lis.Addr().String()))
	if err != nil {
		t.Fatal(err)
	}

	if descriptor.GetFullyQualifiedName() != "test_protos.Outer" {
		t.Fatalf("unexpected message name %s", descriptor.GetFullyQualifiedName())
	}
}
