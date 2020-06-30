package dump

import (
	"sync"
	"time"

	"github.com/bradleyjkemp/grpc-tools/internal"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// recordedServerStream wraps a grpc.ServerStream and allows the dump interceptor to record all sent/received messages
type recordedServerStream struct {
	sync.Mutex
	grpc.ServerStream
	events   []*internal.Message
	headers  metadata.MD
	trailers metadata.MD
}

func (ss *recordedServerStream) SendHeader(headers metadata.MD) error {
	ss.Lock()
	ss.headers = metadata.Join(ss.headers, headers)
	ss.Unlock()
	return ss.ServerStream.SendHeader(headers)
}

func (ss *recordedServerStream) SetHeader(headers metadata.MD) error {
	ss.Lock()
	ss.headers = metadata.Join(ss.headers, headers)
	ss.Unlock()
	return ss.ServerStream.SetHeader(headers)
}

func (ss *recordedServerStream) SetTrailer(trailers metadata.MD) {
	ss.Lock()
	ss.trailers = trailers
	ss.Unlock()
	ss.ServerStream.SetTrailer(trailers)
}

func (ss *recordedServerStream) SendMsg(m interface{}) error {
	message := m.([]byte)
	if message == nil {
		// although the message is nil here, we actually want to save it as the empty message ("")
		message = []byte{}
	}
	ss.Lock()
	ss.events = append(ss.events, &internal.Message{
		MessageOrigin: internal.ServerMessage,
		RawMessage:    message,
		Timestamp:     time.Now(),
	})
	ss.Unlock()
	return ss.ServerStream.SendMsg(m)
}

func (ss *recordedServerStream) RecvMsg(m interface{}) error {
	err := ss.ServerStream.RecvMsg(m)
	if err != nil {
		return err
	}
	// now m is populated
	message := m.(*[]byte)
	ss.Lock()
	ss.events = append(ss.events, &internal.Message{
		MessageOrigin: internal.ClientMessage,
		RawMessage:    *message,
		Timestamp:     time.Now(),
	})
	ss.Unlock()
	return nil
}
