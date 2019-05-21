package internal

import (
	"context"
	"google.golang.org/grpc"
	"sync"
)

type ConnPool struct {
	sync.Mutex
	conns map[string]*grpc.ClientConn
}

func NewConnPool() *ConnPool {
	return &ConnPool{
		conns: map[string]*grpc.ClientConn{},
	}
}

func (c *ConnPool) GetClientConn(ctx context.Context, destination string, dialOptions ...grpc.DialOption) (*grpc.ClientConn, error) {
	c.Lock()
	defer c.Unlock()
	if conn, ok := c.conns[destination]; ok {
		return conn, nil
	}

	conn, err := grpc.DialContext(ctx, destination, dialOptions...)
	if err != nil {
		return nil, err
	}

	c.conns[destination] = conn
	return conn, nil
}
