package codec

type NoopCodec struct{}

func (NoopCodec) Marshal(v interface{}) ([]byte, error) {
	return v.([]byte), nil
}

func (NoopCodec) Unmarshal(data []byte, v interface{}) error {
	*(v.(*[]byte)) = data
	return nil
}

func (NoopCodec) Name() string {
	return "proxy-proto"
}

func (NoopCodec) String() string {
	return "proxy-proto"
}
