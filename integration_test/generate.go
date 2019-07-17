package main

import (
	"encoding/base64"
	"fmt"
	"github.com/golang/protobuf/proto"
)

//go:generate protoc -I . test.proto --go_out=.
func main() {
	for i, val := range []string{
		"ClientRequest",
		"ServerResponse",
		"ServerMessage1",
		"ServerMessage2",
	} {
		marshalled, _ := proto.Marshal(&Outer{
			OuterValue: &Inner{
				InnerValue: val,
				InnerNum:   int64(i + 1),
			},
			OuterNum: int64(i + 1),
		})

		fmt.Println(i, val, base64.StdEncoding.EncodeToString(marshalled))
	}

	marshalled, _ := proto.Marshal(&OuterWithExtra{
		OuterValue: &Inner{
			InnerValue: "ServerMessage2",
			InnerNum:   int64(4 + 1),
		},
		OuterNum:   int64(4 + 1),
		ExtraField: "Detected value",
	})

	fmt.Println(4, "ServerMessage2", base64.StdEncoding.EncodeToString(marshalled))
}
