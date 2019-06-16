package internal

import (
	"context"
	"fmt"
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

func (c *ConnPool) getConn(destination string) (*grpc.ClientConn, bool) {
	c.Lock()
	defer c.Unlock()
	conn, ok := c.conns[destination]
	return conn, ok
}

func (c *ConnPool) addConn(destination string, conn *grpc.ClientConn) {
	c.Lock()
	defer c.Unlock()
	c.conns[destination] = conn
}

func (c *ConnPool) GetClientConn(ctx context.Context, destination string, dialOptions ...grpc.DialOption) (*grpc.ClientConn, error) {
	conn, ok := c.getConn(destination)
	if ok {
		return conn, nil
	}

	conn, err := grpc.DialContext(ctx, destination, dialOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed dialing %s: %v", destination, err)
	}

	c.addConn(destination, conn)
	return conn, nil
}
