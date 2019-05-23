package internal

import (
	"fmt"
	spb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/metadata"
)

type RPC struct {
	Service  string         `json:"service"`
	Method   string         `json:"method"`
	Messages []*StreamEvent `json:"messages"`
	Status   *spb.Status    `json:"error,omitempty"`
	Metadata metadata.MD    `json:"metadata"`
}

func (r RPC) StreamName() string {
	return fmt.Sprintf("/%s/%s", r.Service, r.Method)
}

type MessageOrigin string

const (
	ClientMessage MessageOrigin = "client"
	ServerMessage MessageOrigin = "server"
)

type StreamEvent struct {
	MessageOrigin MessageOrigin `json:"message_origin,omitempty"`
	RawMessage    []byte        `json:"raw_message,omitempty"`
	Message       interface{}   `json:"message,omitempty"`
}
