package grpc_proxy

import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"sync"
)

type connPool struct {
	sync.Mutex
	conns map[string]*grpc.ClientConn
}

func (c *connPool) getClientConn(ctx context.Context, destination string) (*grpc.ClientConn, error) {
	c.Lock()
	defer c.Unlock()
	if conn, ok := c.conns[destination]; ok {
		return conn, nil
	}

	conn, err := grpc.DialContext(ctx, destination,
		// TODO switch TLS based on incoming connection type
		grpc.WithTransportCredentials(credentials.NewTLS(nil)),
		grpc.WithDefaultCallOptions(grpc.ForceCodec(NoopCodec{})),
	)
	if err != nil {
		return nil, err
	}

	c.conns[destination] = conn
	return conn, nil
}
