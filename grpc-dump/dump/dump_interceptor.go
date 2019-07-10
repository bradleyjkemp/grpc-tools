package dump

import (
	"encoding/json"
	"fmt"
	"github.com/bradleyjkemp/grpc-tools/internal"
	"github.com/bradleyjkemp/grpc-tools/internal/proto_decoder"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"io"
	"strings"
)

// dump interceptor implements a gRPC.StreamingServerInterceptor that dumps all RPC details
func dumpInterceptor(output io.Writer, decoder proto_decoder.MessageDecoder) grpc.StreamServerInterceptor {
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

		for _, message := range rpc.Messages {
			message.Message, err = decoder.Decode(info.FullMethod, message)
			// TODO: log warning if error occurs here
		}

		dump, _ := json.Marshal(rpc)
		fmt.Fprintln(output, string(dump))
		return err
	}
}
