package main

import (
	"google.golang.org/grpc"
	"sync"
)

// recordedServerStream wraps a grpc.ServerStream and allows the dump interceptor to record all sent/received messages
type recordedServerStream struct {
	sync.Mutex
	grpc.ServerStream
	events []streamEvent
}

type streamEvent struct {
	ServerMessage []byte `json:"server_message,omitempty"`
	ClientMessage []byte `json:"client_message,omitempty"`
}

func (ss *recordedServerStream) SendMsg(m interface{}) error {
	message := m.([]byte)
	ss.Lock()
	ss.events = append(ss.events, streamEvent{ServerMessage: message})
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
	ss.events = append(ss.events, streamEvent{ClientMessage: *message})
	ss.Unlock()
	return nil
}
