package main

import (
	"encoding/base64"
	"fmt"
	"github.com/bradleyjkemp/grpc-tools/internal/test_protos"
	"github.com/golang/protobuf/proto"
)

//go:generate protoc -I ../internal/test_protos ../internal/test_protos/test.proto --go_out=plugins=grpc:../internal/test_protos
func main() {
	for i, val := range []string{
		"ClientRequest",
		"ServerResponse",
		"ServerMessage1",
		"ServerMessage2",
	} {
		marshalled, _ := proto.Marshal(&test_protos.Outer{
			OuterValue: &test_protos.Inner{
				InnerValue: val,
				InnerNum:   int64(i + 1),
			},
			OuterNum: int64(i + 1),
		})

		fmt.Println(i, val, base64.StdEncoding.EncodeToString(marshalled))
	}

	marshalled, _ := proto.Marshal(&test_protos.OuterWithExtra{
		OuterValue: &test_protos.Inner{
			InnerValue: "ServerMessage2",
			InnerNum:   int64(4 + 1),
		},
		OuterNum:   int64(4 + 1),
		ExtraField: "Detected value",
	})

	fmt.Println(4, "ServerMessage2", base64.StdEncoding.EncodeToString(marshalled))
}
