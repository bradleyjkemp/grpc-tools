package grpc_proxy

import "google.golang.org/grpc"

type noopCodec struct {
	upstream grpc.ClientConn
}

func (noopCodec) Marshal(v interface{}) ([]byte, error) {
	return v.([]byte), nil
}

func (noopCodec) Unmarshal(data []byte, v interface{}) error {
	*(v.(*[]byte)) = data
	return nil
}

func (noopCodec) Name() string {
	return "proxy-proto"
}

func (noopCodec) String() string {
	return "proxy-proto"
}
