package main

import (
	"encoding/json"
	"fmt"
	"github.com/bradleyjkemp/grpc-tools/pkg"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"strings"
)

// dump interceptor implements a gRPC.StreamingServerInterceptor that dumps all RPC details
func dumpInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	dss := &recordedServerStream{ServerStream: ss}
	err := handler(srv, dss)
	rpcStatus, _ := status.FromError(err)

	fullMethod := strings.Split(info.FullMethod, "/")
	md, _ := metadata.FromIncomingContext(ss.Context())
	rpc := pkg.RPC{
		Metadata: md,
		Service:  fullMethod[1],
		Method:   fullMethod[2],
		Messages: dss.events,
		Status:   rpcStatus,
	}

	dump, _ := json.Marshal(rpc)
	fmt.Println(string(dump))
	return err
}
