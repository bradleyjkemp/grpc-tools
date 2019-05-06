package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/bradleyjkemp/grpc-tools/grpc-proxy"
	"github.com/bradleyjkemp/grpc-tools/internal"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"os"
	"time"
)

var (
	destinationServer = flag.String("destination", "", "Destination server to forward requests to. By default the destination is autodetected from the dump metadata.")
	useTLS            = flag.Bool("tls", true, "Whether to use tls when connecting to the server")
	dumpPath          = flag.String("dump", "", "gRPC dump to replay requests from")
)

func main() {
	flag.Parse()
	err := run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		flag.Usage()
		os.Exit(1)
	}
}

func run() error {
	dumpFile, err := os.Open(*dumpPath)
	if err != nil {
		return err
	}

	dumpDecoder := json.NewDecoder(dumpFile)
	for {
		rpc := internal.RPC{}
		err := dumpDecoder.Decode(&rpc)
		if err != nil {
			return fmt.Errorf("failed to decode dump: %s", err)
		}

		conn, err := getConnection(rpc.Metadata)
		if err != nil {
			return fmt.Errorf("failed to connect to destination (%s): %s", *destinationServer, err)
		}
		ctx := metadata.NewOutgoingContext(context.Background(), rpc.Metadata)
		streamName := fmt.Sprintf("/%s/%s", rpc.Service, rpc.Method)
		str, err := conn.NewStream(ctx, &grpc.StreamDesc{
			StreamName:    streamName,
			ServerStreams: true,
			ClientStreams: true,
		}, streamName)
		if err != nil {
			return err
		}

		for _, message := range rpc.Messages {
			if message.RawMessage != nil {
				err := str.SendMsg(message.RawMessage)
				if err != nil {
					return err
				}
			} else {
				var resp []byte
				err := str.RecvMsg(&resp)
				if err != nil {
					return err
				}
				if string(resp) != string(message.ServerMessage) {
					return fmt.Errorf("expected server message != actual server message")
				}
			}
		}
	}
}

var cachedConn *grpc.ClientConn

func getConnection(md metadata.MD) (*grpc.ClientConn, error) {
	if cachedConn != nil {
		return cachedConn, nil
	}

	// if no destination override set then auto-detect from the metadata
	if *destinationServer == "" {
		authority := md.Get(":authority")
		if len(authority) == 0 {
			return nil, fmt.Errorf("no destination override specified and could not auto-detect form dump")
		}

		*destinationServer = authority[0]
	}

	dialCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var conn *grpc.ClientConn
	var err error
	if *useTLS {
		conn, err = grpc.DialContext(dialCtx, *destinationServer,
			grpc.WithTransportCredentials(credentials.NewTLS(nil)),
			grpc.WithDefaultCallOptions(grpc.ForceCodec(grpc_proxy.NoopCodec{})),
			grpc.WithBlock())
	} else {
		conn, err = grpc.DialContext(dialCtx, *destinationServer,
			grpc.WithInsecure(),
			grpc.WithBlock())
	}

	if err != nil {
		return nil, err
	}
	cachedConn = conn
	return cachedConn, nil
}
