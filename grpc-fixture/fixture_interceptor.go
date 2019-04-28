package main

import (
	"encoding/base64"
	"fmt"
	"github.com/bradleyjkemp/grpc-tools/pkg"
	"github.com/davecgh/go-spew/spew"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"
)

type fixtureInterceptor struct {
	allRecordedMethods map[string][]pkg.RPC

	// map of unary request method's request->responses
	unaryMethods map[string]map[string]pkg.RPC
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
	for mapkey, _ := range f.unaryMethods[key] {
		spew.Dump(mapkey)
	}

	response, ok := f.unaryMethods[key][string(receivedMessage)]
	if !ok {
		fmt.Println("huhg?", base64.StdEncoding.EncodeToString(receivedMessage))
		return status.Errorf(codes.Unavailable, "no matching saved response for request %s", string(receivedMessage))
	}
	if response.Status.GetCode() != 0 {
		return status.FromProto(response.Status).Err()
	}

	return ss.SendMsg([]byte(response.Messages[1].ServerMessage))
}
