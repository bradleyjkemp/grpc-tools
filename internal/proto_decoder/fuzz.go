package proto_decoder

import (
	"github.com/bradleyjkemp/grpc-tools/internal/dump_format"
	"github.com/sirupsen/logrus"
)

func Fuzz(data []byte) int {
	dec := NewDecoder(logrus.New())

	_, err := dec.Decode("", &dump_format.Message{
		RawMessage: data,
	})
	if err != nil {
		return 0
	}

	return 1
}
