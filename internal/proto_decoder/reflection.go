package proto_decoder

import (
	"context"
	"github.com/bradleyjkemp/grpc-tools/internal"
	"github.com/bradleyjkemp/grpc-tools/internal/marker"
	"github.com/bradleyjkemp/grpc-tools/internal/proxydialer"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/grpcreflect"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/http/httpproxy"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/grpc/status"
	"strings"
)

type reflectionResolver struct {
	conns     *internal.ConnPool
	blacklist map[string]struct{}
}

func NewReflectionResolver(logger logrus.FieldLogger) *reflectionResolver {
	return &reflectionResolver{
		conns:     internal.NewConnPool(logger, proxydialer.NewProxyDialer(httpproxy.FromEnvironment().ProxyFunc())),
		blacklist: map[string]struct{}{},
	}
}

func (r *reflectionResolver) resolveEncoded(ctx context.Context, fullMethod string, message *internal.Message, md metadata.MD) (*desc.MessageDescriptor, error) {
	authority := md.Get(":authority")
	if len(authority) == 0 {
		return nil, errors.New("no authority in metadata")
	}
	destination := authority[0]

	if _, unimplemented := r.blacklist[destination]; unimplemented {
		return nil, errors.New("destination doesn't implement reflection service")
	}

	options := []grpc.DialOption{
		grpc.WithBlock(),
	}

	if marker.IsTLSRPC(md) {
		options = append(options, grpc.WithTransportCredentials(credentials.NewTLS(nil)))
	} else {
		options = append(options, grpc.WithInsecure())
	}

	conn, err := r.conns.GetClientConn(ctx, destination, options...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to dial server")
	}

	client := grpcreflect.NewClient(ctx, grpc_reflection_v1alpha.NewServerReflectionClient(conn))
	methodParts := strings.Split(fullMethod, "/")
	svc, err := client.ResolveService(methodParts[1])
	if status.Code(err) == codes.Unimplemented {
		r.blacklist[destination] = struct{}{}
		return nil, errors.New("destination doesn't implement reflection service")
	}
	if err != nil {
		return nil, errors.Wrapf(err, "failed to resolve service %s", methodParts[1])
	}

	method := svc.FindMethodByName(methodParts[2])
	if method == nil {
		return nil, errors.Errorf("service descriptor does not contain method %s", methodParts[2])
	}
	switch message.MessageOrigin {
	case internal.ClientMessage:
		return method.GetInputType(), nil
	case internal.ServerMessage:
		return method.GetOutputType(), nil
	default:
		return nil, errors.New("invalid message origin")
	}
}

func (r *reflectionResolver) resolveDecoded(fullMethod string, message *internal.Message, md metadata.MD) (*desc.MessageDescriptor, error) {
	panic("implement me")
}
