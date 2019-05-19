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

func (c *connPool) getClientConn(ctx context.Context, destination string, tls bool) (*grpc.ClientConn, error) {
	c.Lock()
	defer c.Unlock()
	if conn, ok := c.conns[destination]; ok {
		return conn, nil
	}

	options := []grpc.DialOption{grpc.WithDefaultCallOptions(grpc.ForceCodec(NoopCodec{}))}

	if tls {
		options = append(options, grpc.WithTransportCredentials(credentials.NewTLS(nil)))
	} else {
		options = append(options, grpc.WithInsecure())
	}

	conn, err := grpc.DialContext(ctx, destination, options...)
	if err != nil {
		return nil, err
	}

	c.conns[destination] = conn
	return conn, nil
}
