package test_server

import (
	"context"
	"github.com/bradleyjkemp/grpc-tools/internal/test_protos"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"net"
)

//go:generate protoc -I ../internal/test_protos ../internal/test_protos/test.proto --go_out=plugins=grpc:../internal/test_protos
func Listen(ctx context.Context, lis net.Listener) (err error) {
	s := grpc.NewServer()
	reflection.Register(s)
	test_protos.RegisterTestServiceServer(s, testServiceImpl{})
	go func() {
		err = s.Serve(lis)
	}()
	<-ctx.Done()
	s.GracefulStop()
	return
}

type testServiceImpl struct{}

func (testServiceImpl) TestUnaryClientRequest(context.Context, *test_protos.Outer) (*test_protos.Outer, error) {
	return nil, status.Error(codes.Unimplemented, "stub")
}

func (testServiceImpl) TestStreamingServerMessages(_ *test_protos.Outer, s test_protos.TestService_TestStreamingServerMessagesServer) error {
	return status.Error(codes.Unimplemented, "stub")
}
