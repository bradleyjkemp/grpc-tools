package proto_decoder

import (
	"github.com/bradleyjkemp/grpc-tools/internal"
	"github.com/sirupsen/logrus"
)

func Fuzz(data []byte) int {
	dec := NewDecoder(logrus.New())

	_, err := dec.Decode("", &internal.Message{
		RawMessage: data,
	}, nil)
	if err != nil {
		return 0
	}

	return 1
}
