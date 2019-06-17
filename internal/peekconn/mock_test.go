package peekconn

import (
	"bytes"
	"net"
	"time"
)

type mockConn struct {
	*bytes.Buffer
}

func (mockConn) Close() error {
	panic("implement me")
}

func (mockConn) LocalAddr() net.Addr {
	panic("implement me")
}

func (mockConn) RemoteAddr() net.Addr {
	panic("implement me")
}

func (mockConn) SetDeadline(t time.Time) error {
	panic("implement me")
}

func (mockConn) SetReadDeadline(t time.Time) error {
	panic("implement me")
}

func (mockConn) SetWriteDeadline(t time.Time) error {
	panic("implement me")
}
