package internal

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"net"
	"sync"
)

type contextDialer = func(context.Context, string) (net.Conn, error)

type ConnPool struct {
	sync.Mutex
	conns  map[string]*grpc.ClientConn
	logger logrus.FieldLogger
	dialer contextDialer
}

func NewConnPool(logger logrus.FieldLogger, dialer contextDialer) *ConnPool {
	return &ConnPool{
		conns:  map[string]*grpc.ClientConn{},
		logger: logger.WithField("", "connpool"),
		dialer: dialer,
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
		c.logger.Debugf("Returning cached connection to %s", destination)
		return conn, nil
	}

	c.logger.Debugf("Dialing new connection to %s", destination)
	dialOptions = append(dialOptions, grpc.WithContextDialer(c.dialer))
	conn, err := grpc.DialContext(ctx, destination, dialOptions...)
	if err != nil {
		c.logger.WithError(err).Debugf("Failed dialing to %s", destination)
		return nil, fmt.Errorf("failed dialing %s: %v", destination, err)
	}

	c.addConn(destination, conn)
	return conn, nil
}
