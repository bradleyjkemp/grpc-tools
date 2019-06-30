package main

import (
	"encoding/json"
	"fmt"
	"github.com/bradleyjkemp/grpc-tools/internal"
	"github.com/bradleyjkemp/grpc-tools/internal/proto_decoder"
	"github.com/golang/protobuf/proto"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"os"
	"strings"
)

// dump interceptor implements a gRPC.StreamingServerInterceptor that dumps all RPC details
func dumpInterceptor(knownMethods map[string]*desc.MethodDescriptor) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		dss := &recordedServerStream{ServerStream: ss}
		err := handler(srv, dss)
		var rpcStatus *internal.Status
		if err != nil {
			grpcStatus, _ := status.FromError(err)
			rpcStatus = &internal.Status{
				Code:    grpcStatus.Code().String(),
				Message: grpcStatus.Message(),
			}
		}

		fullMethod := strings.Split(info.FullMethod, "/")
		md, _ := metadata.FromIncomingContext(ss.Context())
		rpc := internal.RPC{
			Service:  fullMethod[1],
			Method:   fullMethod[2],
			Messages: dss.events,
			Status:   rpcStatus,
			Metadata: md,
		}

		knownMethod := knownMethods[info.FullMethod]
		for _, message := range rpc.Messages {
			var dyn *dynamic.Message
			if knownMethod == nil {
				dec := proto_decoder.NewDecoder(proto_decoder.NewUnknownResolver())
				dyn, err = dec.Decode(info.FullMethod, message.RawMessage)
			} else {
				// have proper type information so e.g. can have field names in the text representation
				switch message.MessageOrigin {
				case internal.ClientMessage:
					dyn = dynamic.NewMessage(knownMethod.GetInputType())
				case internal.ServerMessage:
					dyn = dynamic.NewMessage(knownMethod.GetOutputType())
				}
			}
			err = proto.Unmarshal(message.RawMessage, dyn)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to unmarshal message: %v\n", err)
			}

			message.Message = dyn
		}

		dump, _ := json.Marshal(rpc)
		fmt.Println(string(dump))
		return err
	}
}
