package proto_decoder

import (
	"github.com/bradleyjkemp/grpc-tools/internal"
)

func Fuzz(data []byte) int {
	dec := NewDecoder()

	_, err := dec.Decode("", &internal.Message{
		RawMessage: data,
	})
	if err != nil {
		return 0
	}

	return 1
}
