package pkg

import (
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type RPC struct {
	Service  string         `json:"service"`
	Method   string         `json:"method"`
	Messages []StreamEvent  `json:"messages"`
	Status   *status.Status `json:"error"`
	Metadata metadata.MD    `json:"metadata"`
}

type StreamEvent struct {
	ServerMessage []byte `json:"server_message,omitempty"`
	ClientMessage []byte `json:"client_message,omitempty"`
}
