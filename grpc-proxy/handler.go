package grpc_proxy

import (
	"context"
	"fmt"
	"github.com/bradleyjkemp/grpc-tools/internal/tls"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"io"
	"os"
)

// Originally based on github.com/mwitkow/grpc-proxy/proxy/handler.go
func (s *server) proxyHandler(srv interface{}, ss grpc.ServerStream) error {
	md, ok := metadata.FromIncomingContext(ss.Context())
	if !ok {
		return status.Error(codes.Unknown, "could not extract metadata from request")
	}

	authority := md.Get(":authority")
	var destinationAddr string
	switch {
	case len(authority) > 0:
		// use authority from request
		destinationAddr = authority[0]
	case s.destination != "":
		// fallback to hardcoded destination (used by clients not supporting HTTP proxies)
		destinationAddr = s.destination
	default:
		// no destination can be determined so just error
		return status.Error(codes.Unimplemented, "no proxy destination configured")
	}

	options := []grpc.DialOption{
		grpc.WithDefaultCallOptions(grpc.ForceCodec(NoopCodec{})),
		grpc.WithBlock(),
	}
	if tls.IsTLSRPC(md) {
		options = append(options, grpc.WithTransportCredentials(credentials.NewTLS(nil)))
	} else {
		options = append(options, grpc.WithInsecure())
	}

	destination, err := s.connPool.GetClientConn(ss.Context(), destinationAddr, options...)
	if err != nil {
		return err
	}

	// little bit of gRPC internals never hurt anyone
	fullMethodName, ok := grpc.MethodFromServerStream(ss)
	if !ok {
		return status.Errorf(codes.Internal, "no method exists in context")
	}

	clientCtx, clientCancel := getClientCtx(ss.Context())
	clientStream, err := destination.NewStream(clientCtx, proxyStreamDesc, fullMethodName)
	if err != nil {
		return err
	}

	// Explicitly *do not close* s2cErrChan and c2sErrChan, otherwise the select below will not terminate.
	// Channels do not have to be closed, it is just a control flow mechanism, see
	// https://groups.google.com/forum/#!msg/golang-nuts/pZwdYRGxCIk/qpbHxRRPJdUJ
	s2cErrChan := forwardServerToClient(ss, clientStream)
	c2sErrChan := forwardClientToServer(clientStream, ss)
	// We don't know which side is going to stop sending first, so we need a select between the two.
	for i := 0; i < 2; i++ {
		select {
		case s2cErr := <-s2cErrChan:
			if s2cErr == io.EOF {
				// this is the happy case where the sender has encountered io.EOF, and won't be sending anymore./
				// the clientStream>serverStream may continue pumping though.
				clientStream.CloseSend()
				break
			} else {
				// however, we may have gotten a receive error (stream disconnected, a read error etc) in which case we need
				// to cancel the clientStream to the backend, let all of its goroutines be freed up by the CancelFunc and
				// exit with an error to the stack
				clientCancel()
				fmt.Fprintln(os.Stderr, "failed proxying s2c", s2cErr)
				return grpc.Errorf(codes.Internal, "failed proxying s2c: %v", s2cErr)
			}
		case c2sErr := <-c2sErrChan:
			// This happens when the clientStream has nothing else to offer (io.EOF), returned a gRPC error. In those two
			// cases we may have received Trailers as part of the call. In case of other errors (stream closed) the trailers
			// will be nil.
			ss.SetTrailer(clientStream.Trailer())
			// c2sErr will contain RPC error from client code. If not io.EOF return the RPC error as server stream error.
			if c2sErr != io.EOF {
				return c2sErr
			}
			return nil
		}
	}
	return grpc.Errorf(codes.Internal, "gRPC proxying should never reach this stage.")
}

func getClientCtx(serverCtx context.Context) (context.Context, context.CancelFunc) {
	clientCtx, clientCancel := context.WithCancel(serverCtx)

	md, ok := metadata.FromIncomingContext(serverCtx)
	if ok {
		clientCtx = metadata.NewOutgoingContext(clientCtx, md)
	}

	return clientCtx, clientCancel
}

func forwardClientToServer(src grpc.ClientStream, dst grpc.ServerStream) chan error {
	ret := make(chan error, 1)
	go func() {
		var f []byte
		for i := 0; ; i++ {
			if err := src.RecvMsg(&f); err != nil {
				ret <- err // this can be io.EOF which is happy case
				break
			}
			if i == 0 {
				// This is a bit of a hack, but client to server headers are only readable after first client msg is
				// received but must be written to server stream before the first msg is flushed.
				// This is the only place to do it nicely.
				md, err := src.Header()
				if err != nil {
					ret <- err
					break
				}
				if err := dst.SendHeader(md); err != nil {
					ret <- err
					break
				}
			}
			if err := dst.SendMsg(f); err != nil {
				ret <- err
				break
			}
		}
	}()
	return ret
}

func forwardServerToClient(src grpc.ServerStream, dst grpc.ClientStream) chan error {
	ret := make(chan error, 1)
	go func() {
		var f []byte
		for i := 0; ; i++ {
			if err := src.RecvMsg(&f); err != nil {
				ret <- err // this can be io.EOF which is happy case
				break
			}
			if err := dst.SendMsg(f); err != nil {
				ret <- err
				break
			}
		}
	}()
	return ret
}
