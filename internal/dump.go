package internal

import (
	spb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/metadata"
)

type RPC struct {
	Service  string         `json:"service"`
	Method   string         `json:"method"`
	Messages []*StreamEvent `json:"messages"`
	Status   *spb.Status    `json:"error"`
	Metadata metadata.MD    `json:"metadata"`
}

type messageOrigin string

const (
	ClientMessage messageOrigin = "client"
	ServerMessage messageOrigin = "server"
)

type StreamEvent struct {
	MessageOrigin messageOrigin `json:"message_origin,omitempty"`
	RawMessage    []byte        `json:"raw_message,omitempty"`
	Message       interface{}   `json:"message,omitempty"`
}
