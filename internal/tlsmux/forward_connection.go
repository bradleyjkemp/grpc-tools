package tlsmux

import (
	"io"
	"net"
	"sync"
)

type tcpLike interface {
	CloseRead() error
	CloseWrite() error
}

func forwardConnection(conn net.Conn, destConn net.Conn) error {
	if isTCPTunnel(conn, destConn) {
		// each side of the connection can be closed independently so no synchronisation required
		go copyAndCloseTCP(conn, destConn)
		go copyAndCloseTCP(destConn, conn)
		return nil
	}

	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		io.Copy(conn, destConn)
		wg.Done()
	}()
	go func() {
		io.Copy(destConn, conn)
		wg.Done()
	}()
	go func() {
		wg.Wait()
		conn.Close()
		destConn.Close()
	}()
	return nil
}

func copyAndCloseTCP(dst, src net.Conn) {
	io.Copy(dst, src)
	dst.(tcpLike).CloseWrite()
	src.(tcpLike).CloseRead()
}

// checks if the two connections are "TCP-like"
// i.e. the two connection halves can be close separately
func isTCPTunnel(a, b net.Conn) bool {
	_, aTCP := a.(tcpLike)
	_, bTCP := b.(tcpLike)
	return aTCP && bTCP
}

// The following can be used to debug all traffic forwarded over a connection
//
//type debugReader struct {
//	io.Reader
//}
//
//func (d *debugReader) Read(p []byte) (n int, err error) {
//	n, err = d.Reader.Read(p)
//	fmt.Printf("[%p] Read bytes: %v, %v\n", d, string(p[:n]), err)
//	return n, err
//}
