package dump_format

import (
	"fmt"
	"google.golang.org/grpc/metadata"
	"time"
)

type LineType string

const (
	RPCLine     LineType = "rpc"
	MessageLine LineType = "message"
	StatusLine  LineType = "status"
)

type RPC struct {
	Timestamp time.Time   `json:"timestamp"`
	Type      LineType    `json:"type"`
	ID        int         `json:"id"`
	Service   string      `json:"service"`
	Method    string      `json:"method"`
	Metadata  metadata.MD `json:"metadata"`
}

func (r RPC) StreamName() string {
	return fmt.Sprintf("/%s/%s", r.Service, r.Method)
}

type Message struct {
	Timestamp     time.Time     `json:"timestamp"`
	Type          LineType      `json:"type"`
	ID            int           `json:"id"`
	MessageID     int           `json:"message_id"`
	MessageOrigin MessageOrigin `json:"origin,omitempty"`
	RawMessage    []byte        `json:"raw,omitempty"`
	Message       interface{}   `json:"message,omitempty"`
}

type Status struct {
	Timestamp time.Time `json:"timestamp"`
	Type      LineType  `json:"type"`
	ID        int       `json:"id"`
	Code      string    `json:"code"`
	Message   string    `json:"message,omitempty"`
}

type MessageOrigin string

const (
	ClientMessage MessageOrigin = "client"
	ServerMessage MessageOrigin = "server"
)
