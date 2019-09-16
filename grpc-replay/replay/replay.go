package replay

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/bradleyjkemp/grpc-tools/grpc-proxy"
	"github.com/bradleyjkemp/grpc-tools/internal"
	"github.com/bradleyjkemp/grpc-tools/internal/codec"
	"github.com/bradleyjkemp/grpc-tools/internal/dump_format"
	"github.com/bradleyjkemp/grpc-tools/internal/marker"
	"github.com/bradleyjkemp/grpc-tools/internal/proto_decoder"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"io"
	"os"
	"strings"
	"time"
)

func Run(protoRoots, protoDescriptors, dumpPath, destinationOverride string, dialer grpc_proxy.ContextDialer) error {
	pool := internal.NewConnPool(logrus.New(), dialer)

	dumpFile, err := os.Open(dumpPath)
	if err != nil {
		return err
	}
	var resolvers []proto_decoder.MessageResolver
	if protoRoots != "" {
		r, err := proto_decoder.NewFileResolver(strings.Split(protoRoots, ",")...)
		if err != nil {
			return err
		}
		resolvers = append(resolvers, r)
	}
	if protoDescriptors != "" {
		r, err := proto_decoder.NewDescriptorResolver(strings.Split(protoRoots, ",")...)
		if err != nil {
			return err
		}
		resolvers = append(resolvers, r)
	}
	encoder := proto_decoder.NewEncoder(resolvers...)

	rpcs := map[int64]grpc.ClientStream{}
	streamNames := map[int64]string{}

	dumpDecoder := json.NewDecoder(dumpFile)
RPC:
	for {
		line := &dump_format.DecodedLine{}
		err := dumpDecoder.Decode(line)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		switch l := line.Get().(type) {
		case *dump_format.RPC:
			fmt.Println("")
			// First time we've seen an RPC so start a new stream
			conn, err := getConnection(pool, l.Metadata, destinationOverride)
			if err != nil {
				return fmt.Errorf("failed to connect to destination (%s): %s", destinationOverride, err)
			}

			// RPC has metadata added by grpc-dump that should be removed before sending
			// (so that we're sending as close as possible to the original request)
			marker.RemoveHTTPSMarker(l.Metadata)

			ctx := metadata.NewOutgoingContext(context.Background(), l.Metadata)
			streamName := l.StreamName()
			str, err := conn.NewStream(ctx, &grpc.StreamDesc{
				StreamName:    streamName,
				ServerStreams: true,
				ClientStreams: true,
			}, streamName)
			if err != nil {
				return fmt.Errorf("failed to make new stream: %v", err)
			}
			rpcs[l.ID] = str
			streamNames[l.ID] = streamName
			fmt.Print(streamName, "...")

		case *dump_format.Message:
			str := rpcs[l.ID]
			msgBytes, err := encoder.Encode(streamNames[l.ID], l)
			if err != nil {
				return fmt.Errorf("failed to encode message: %v", err)
			}

			switch l.MessageOrigin {
			case dump_format.ClientMessage:
				err := str.SendMsg(msgBytes)
				if err != nil {
					return fmt.Errorf("failed to send message: %v", err)
				}
			case dump_format.ServerMessage:
				var resp []byte
				err := str.RecvMsg(&resp)
				if err != nil {
					// TODO when do we assert on RPC errors?
					return fmt.Errorf("failed to recv message: %v", err)
				}
				if string(resp) != string(msgBytes) {
					fmt.Println("Err mismatch")
					continue RPC
				}
			default:
				return fmt.Errorf("invalid message type: %v", l.MessageOrigin)
			}

		case *dump_format.Status:
			fmt.Println("OK")
			// TODO: handle RPC statuses (issue #24)

		default:
			return errors.Errorf("unknown line type %T", line)
		}
	}

	return nil
}

func getConnection(pool *internal.ConnPool, md metadata.MD, destinationOverride string) (*grpc.ClientConn, error) {
	// if no destination override set then auto-detect from the metadata
	var destination = destinationOverride
	if destination == "" {
		authority := md.Get(":authority")
		if len(authority) == 0 {
			return nil, fmt.Errorf("no destination override specified and could not auto-detect from dump")
		}
		destination = authority[0]
	}

	options := []grpc.DialOption{
		grpc.WithDefaultCallOptions(grpc.ForceCodec(codec.NoopCodec{})),
		grpc.WithBlock(),
	}

	if marker.IsTLSRPC(md) {
		options = append(options, grpc.WithTransportCredentials(credentials.NewTLS(nil)))
	} else {
		options = append(options, grpc.WithInsecure())
	}

	dialCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return pool.GetClientConn(dialCtx, destination, options...)
}
