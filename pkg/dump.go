package pkg

import (
	spb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/metadata"
)

type RPC struct {
	Service  string        `json:"service"`
	Method   string        `json:"method"`
	Messages []StreamEvent `json:"messages"`
	Status   *spb.Status   `json:"error"`
	Metadata metadata.MD   `json:"metadata"`
}

type StreamEvent struct {
	ServerMessage []byte `json:"server_message,omitempty"`
	ClientMessage []byte `json:"client_message,omitempty"`
}
