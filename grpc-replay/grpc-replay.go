package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/bradleyjkemp/grpc-tools/internal"
	"github.com/bradleyjkemp/grpc-tools/internal/codec"
	"github.com/bradleyjkemp/grpc-tools/internal/marker"
	"github.com/bradleyjkemp/grpc-tools/internal/proto_decoder"
	_ "github.com/bradleyjkemp/grpc-tools/internal/versionflag"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"io"
	"os"
	"strings"
	"time"
)

var (
	destinationOverride = flag.String("destination", "", "Destination server to forward requests to. By default the destination for each RPC is autodetected from the dump metadata.")
	dumpPath            = flag.String("dump", "", "The gRPC dump to replay requests from")
	protoRoots          = flag.String("proto_roots", "", "A comma separated list of directories to search for gRPC service definitions.")
	protoDescriptors    = flag.String("proto_descriptors", "", "A comma separated list of proto descriptors to load gRPC service definitions from.")
)

func main() {
	flag.Parse()
	err := run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		flag.Usage()
		os.Exit(1)
	}
}

func run() error {
	dumpFile, err := os.Open(*dumpPath)
	if err != nil {
		return err
	}
	var resolvers []proto_decoder.MessageResolver
	if *protoRoots != "" {
		r, err := proto_decoder.NewFileResolver(strings.Split(*protoRoots, ",")...)
		if err != nil {
			return err
		}
		resolvers = append(resolvers, r)
	}
	if *protoDescriptors != "" {
		r, err := proto_decoder.NewDescriptorResolver(strings.Split(*protoRoots, ",")...)
		if err != nil {
			return err
		}
		resolvers = append(resolvers, r)
	}
	encoder := proto_decoder.NewEncoder(resolvers...)

	dumpDecoder := json.NewDecoder(dumpFile)
RPC:
	for {
		rpc := internal.RPC{}
		err := dumpDecoder.Decode(&rpc)
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to decode dump: %s", err)
		}

		conn, err := getConnection(rpc.Metadata)
		if err != nil {
			return fmt.Errorf("failed to connect to destination (%s): %s", *destinationOverride, err)
		}

		// RPC has metadata added by grpc-dump that should be removed before sending
		// (so that we're sending as close as possible to the original request)
		marker.RemoveHTTPSMarker(rpc.Metadata)

		ctx := metadata.NewOutgoingContext(context.Background(), rpc.Metadata)
		streamName := rpc.StreamName()
		str, err := conn.NewStream(ctx, &grpc.StreamDesc{
			StreamName:    streamName,
			ServerStreams: true,
			ClientStreams: true,
		}, streamName)
		if err != nil {
			return fmt.Errorf("failed to make new stream: %v", err)
		}

		fmt.Print(streamName, "...")
		for _, message := range rpc.Messages {
			var msgBytes []byte
			switch {
			case message.Message == nil && message.RawMessage != nil:
				msgBytes = message.RawMessage

			case message.Message != nil:
				msgBytes, err = encoder.Encode(streamName, message.MessageOrigin, message.Message)
				if err != nil {
					// TODO: add warning here
					msgBytes = message.RawMessage
				}

			case message.Message == nil && message.RawMessage == nil:
				return fmt.Errorf("no message available: both Message and RawMessage are nil")
			}

			switch message.MessageOrigin {
			case internal.ClientMessage:
				err := str.SendMsg(msgBytes)
				if err != nil {
					return fmt.Errorf("failed to send message: %v", err)
				}
			case internal.ServerMessage:
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
				return fmt.Errorf("invalid message type: %v", message.MessageOrigin)
			}
		}
		fmt.Println("OK")
	}
	return nil
}

var cachedConns = internal.NewConnPool(logrus.New())

func getConnection(md metadata.MD) (*grpc.ClientConn, error) {
	// if no destination override set then auto-detect from the metadata
	var destination = *destinationOverride
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
	return cachedConns.GetClientConn(dialCtx, destination, options...)
}
