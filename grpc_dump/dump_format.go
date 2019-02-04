package main

import (
	"google.golang.org/grpc/metadata"
)

type rpc struct {
	Service string `json:"service"`
	Method string `json:"method"`
	Messages []streamEvent `json:"messages"`
	Error error `json:"error"`
	Metadata metadata.MD `json:"metadata"`
}
