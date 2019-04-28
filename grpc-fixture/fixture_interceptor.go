package main

import (
	"github.com/bradleyjkemp/grpc-tools/pkg"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"
)

type fixtureInterceptor struct {
	allRecordedMethods map[string][][]pkg.StreamEvent

	// map of unary request method's request->responses
	unaryMethods map[string]map[string]string
}

// fixtureInterceptor implements a gRPC.StreamingServerInterceptor that replays saved responses
func (f *fixtureInterceptor) intercept(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, _ grpc.StreamHandler) error {
	fullMethod := strings.Split(info.FullMethod, "/")
	key := fullMethod[1] + "/" + fullMethod[2]

	if len(f.allRecordedMethods[key]) == 0 {
		return status.Error(codes.Unavailable, "no saved responses found for method")
	}

	if len(f.unaryMethods[key]) == 0 {
		return status.Error(codes.Unimplemented, "non-unary methods not yet implemented")
	}

	var receivedMessage []byte
	err := ss.RecvMsg(&receivedMessage)
	if err != nil {
		return err
	}
	response, ok := f.unaryMethods[key][string(receivedMessage)]
	if !ok {
		return status.Error(codes.Unavailable, "no matching saved response for request")
	}
	return ss.SendMsg([]byte(response))
}
