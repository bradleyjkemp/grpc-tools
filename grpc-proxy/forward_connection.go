package grpc_proxy

import (
	"io"
	"net"
)

func forwardConnection(proxConn proxiedConn) error {
	destinationConn, err := net.Dial(proxConn.LocalAddr().Network(), proxConn.originalDestination)
	if err != nil {
		return err
	}
	destConnTCP := destinationConn.(*net.TCPConn)
	go copyAndClose(proxConn, destConnTCP)
	go copyAndClose(destConnTCP, proxConn)
	return nil
}

func copyAndClose(dst, src bidirectionalConn) {
	io.Copy(dst, src)
	dst.CloseWrite()
	src.CloseRead()
}
