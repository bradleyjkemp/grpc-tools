package dump

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/bradleyjkemp/grpc-tools/internal"
	"github.com/bradleyjkemp/grpc-tools/internal/proto_decoder"
	"github.com/bradleyjkemp/grpc-tools/internal/proto_descriptor"
	"github.com/golang/protobuf/jsonpb"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// dump interceptor implements a gRPC.StreamingServerInterceptor that dumps all RPC details
var mu sync.Mutex

type pbm struct {
	*dynamic.Message
}

func (p *pbm) MarshalJSON() ([]byte, error) {
	fd := make([]*desc.FileDescriptor, 0)
	proto_descriptor.MsgDesc.Lock()
	defer proto_descriptor.MsgDesc.Unlock()
	for _, d := range proto_descriptor.MsgDesc.Desc {
		fd = append(fd, d.GetFile())
	}
	return p.MarshalJSONPB(
		&jsonpb.Marshaler{
			AnyResolver: dynamic.AnyResolver(
				dynamic.NewMessageFactoryWithDefaults(),
				fd...,
			),
		})
}

func dumpInterceptor(logger logrus.FieldLogger, output io.Writer, decoder proto_decoder.MessageDecoder) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		dss := &recordedServerStream{ServerStream: ss}
		rpcErr := handler(srv, dss)
		var rpcStatus *internal.Status
		if rpcErr != nil {
			grpcStatus, _ := status.FromError(rpcErr)
			rpcStatus = &internal.Status{
				Code:    grpcStatus.Code().String(),
				Message: grpcStatus.Message(),
			}
		}

		fullMethod := strings.Split(info.FullMethod, "/")
		md, _ := metadata.FromIncomingContext(ss.Context())
		dss.Lock()
		rpc := internal.RPC{
			Service:              fullMethod[1],
			Method:               fullMethod[2],
			Messages:             dss.events,
			Status:               rpcStatus,
			Metadata:             md,
			MetadataRespHeaders:  dss.headers,
			MetadataRespTrailers: dss.trailers,
		}

		var err error
		for i := range rpc.Messages {
			msg, err := decoder.Decode(info.FullMethod, rpc.Messages[i])
			if err != nil {
				logger.WithError(err).Warn("Failed to decode message")
			}
			rpc.Messages[i].Message = &pbm{msg}
		}
		dump, err := json.Marshal(rpc)
		if err != nil {
			logger.WithError(err).Fatal("Failed to marshal rpc")
		}
		fmt.Fprintln(output, string(dump))
		dss.Unlock()
		return rpcErr
	}
}
