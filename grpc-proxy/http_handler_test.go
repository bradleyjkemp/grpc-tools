package grpc_proxy

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

var (
	testDestination = "test-url.example.com"
)

type stubGRPCWebHandler struct {
	handler http.HandlerFunc
	isGRPC  func(req *http.Request) bool
}

func (s stubGRPCWebHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	s.handler(resp, req)
}

func (s stubGRPCWebHandler) IsGrpcWebRequest(req *http.Request) bool {
	return s.isGRPC(req)
}

func TestHTTPHandler_RedirectsCONNECT(t *testing.T) {
	logger := logrus.New()
	proxiedConn := make(chan net.Conn, 1)
	destination := make(chan string, 1)
	handlerFinished := sync.WaitGroup{}
	handlerFinished.Add(1)
	s := httptest.NewServer(newHttpServer(logger, nil, func(conn net.Conn, dest string) {
		proxiedConn <- conn
		destination <- dest
		handlerFinished.Done()
	}, nil).Handler)

	clientConn, err := net.Dial(s.Listener.Addr().Network(), s.Listener.Addr().String())
	if err != nil {
		panic(err)
	}
	_, err = fmt.Fprintf(clientConn, "CONNECT %s HTTP/1.1\n\n", testDestination)
	if err != nil {
		panic(err)
	}

	// Must get 200 OK response
	expectedResponse := "HTTP/1.1 200 OK\r\n\r\n"
	resp := make([]byte, len(expectedResponse))
	n, err := clientConn.Read(resp)
	if err != nil {
		panic(err)
	}
	if n != len(expectedResponse) || string(resp) != expectedResponse {
		panic(string(resp))
	}

	handlerFinished.Wait()

	select {
	case dest := <-destination:
		if dest != testDestination {
			panic(dest)
		}
	default:
		panic("No destination proxied")
	}

	select {
	case <-proxiedConn:
	default:
		panic("No connection proxied")
	}
}

func TestHTTPHandler_InterceptsGRPC(t *testing.T) {
	logger := logrus.New()
	var gRPCHandlerCalled bool
	s := httptest.NewServer(newHttpServer(logger, stubGRPCWebHandler{
		handler: func(_ http.ResponseWriter, _ *http.Request) {
			gRPCHandlerCalled = true
		},
		isGRPC: func(_ *http.Request) bool {
			return true
		},
	}, nil, nil).Handler)

	_, err := http.Post(s.URL, "", nil)
	if err != nil {
		panic(err)
	}
	if !gRPCHandlerCalled {
		panic("gRPC Handler not called")
	}
}

func TestHTTPHandler_ReverseProxiesUnknown(t *testing.T) {
	logger := logrus.New()
	var reverseProxyCalled bool
	s := httptest.NewServer(newHttpServer(logger, stubGRPCWebHandler{
		handler: func(_ http.ResponseWriter, _ *http.Request) {
			panic("grpc handler called")
		},
		isGRPC: func(_ *http.Request) bool {
			return false
		},
	}, nil, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		reverseProxyCalled = true
	})).Handler)

	_, err := http.Get(s.URL)
	if err != nil {
		panic(err)
	}
	if !reverseProxyCalled {
		panic("gRPC Handler not called")
	}
}
