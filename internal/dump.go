package internal

import (
	"fmt"
	"time"

	"google.golang.org/grpc/metadata"
)

type RPC struct {
	Service              string      `json:"service"`
	Method               string      `json:"method"`
	Messages             []*Message  `json:"messages"`
	Status               *Status     `json:"error,omitempty"`
	Metadata             metadata.MD `json:"metadata"`
	MetadataRespHeaders  metadata.MD `json:"metadata_response_headers"`
	MetadataRespTrailers metadata.MD `json:"metadata_response_trailers"`
}

type Status struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (r RPC) StreamName() string {
	return fmt.Sprintf("/%s/%s", r.Service, r.Method)
}

type MessageOrigin string

const (
	ClientMessage MessageOrigin = "client"
	ServerMessage MessageOrigin = "server"
)

type Message struct {
	MessageOrigin MessageOrigin `json:"message_origin,omitempty"`
	RawMessage    []byte        `json:"raw_message,omitempty"`
	Message       interface{}   `json:"message,omitempty"`
	Timestamp     time.Time     `json:"timestamp"`
}
