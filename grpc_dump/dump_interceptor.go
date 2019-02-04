package main

import (
	"encoding/json"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"strings"
)

// dump interceptor implements a gRPC.StreamingServerInterceptor that dumps all RPC details
func dumpInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	dss := &recordedServerStream{ServerStream: ss}
	err := handler(srv, dss)

	fullMethod := strings.Split(info.FullMethod, "/")
	md, _ := metadata.FromIncomingContext(ss.Context())
	rpc := rpc{
		Metadata: md,
		Service:  fullMethod[1],
		Method:   fullMethod[2],
		Messages: dss.events,
		Error: err,
	}

	dump, _ := json.Marshal(rpc)
	fmt.Println(string(dump))
	return err
}
