package dump

import (
	"encoding/json"
	"fmt"
	"github.com/bradleyjkemp/grpc-tools/internal/dump_format"
	"github.com/bradleyjkemp/grpc-tools/internal/proto_decoder"
	"github.com/sirupsen/logrus"
	"go.uber.org/atomic"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"io"
	"strings"
	"time"
)

var rpcID = atomic.NewInt64(-1)

// dump interceptor implements a gRPC.StreamingServerInterceptor that dumps all RPC details
func dumpInterceptor(logger logrus.FieldLogger, output io.Writer, decoder proto_decoder.MessageDecoder) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		fullMethod := strings.Split(info.FullMethod, "/")
		md, _ := metadata.FromIncomingContext(ss.Context())
		rpc := dump_format.RPC{
			Timestamp: time.Now(),
			Type:      dump_format.RPCLine,
			ID:        rpcID.Inc(),
			Service:   fullMethod[1],
			Method:    fullMethod[2],
			Metadata:  md,
		}
		dump, _ := json.Marshal(rpc)
		fmt.Fprintln(output, string(dump))

		dss := &dumpServerStream{
			ServerStream: ss,
			rpcID:        rpc.ID,
			fullMethod:   info.FullMethod,
			output:       output,
			logger:       logger,
			decoder:      decoder,
		}
		rpcErr := handler(srv, dss)

		rpcStatus := &dump_format.Status{
			Timestamp: time.Now(),
			Type:      dump_format.StatusLine,
			ID:        rpc.ID,
		}
		if rpcErr != nil {
			grpcStatus, _ := status.FromError(rpcErr)
			rpcStatus.Code = grpcStatus.Code().String()
			rpcStatus.Message = grpcStatus.Message()
		} else {
			rpcStatus.Code = codes.OK.String()
		}
		statusDump, _ := json.Marshal(rpcStatus)
		fmt.Fprintln(output, string(statusDump))

		return rpcErr
	}
}
