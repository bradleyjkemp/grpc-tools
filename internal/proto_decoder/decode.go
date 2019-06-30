package proto_decoder

import (
	"github.com/golang/protobuf/proto"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
)

type messageResolver interface {
	resolve(fullMethod string, raw []byte) (*desc.MessageDescriptor, error)
}

type messageDecoder struct {
	resolvers []messageResolver
}

// Chain together a number of resolvers to decode incoming messages.
// Resolvers are in priority order, the first to return a nil error
// is used to decode the message.
func NewDecoder(resolvers ...messageResolver) *messageDecoder {
	return &messageDecoder{
		resolvers: resolvers,
	}
}

func (d *messageDecoder) Decode(fullMethod string, raw []byte) (*dynamic.Message, error) {
	var err error
	for _, resolver := range d.resolvers {
		var descriptor *desc.MessageDescriptor
		descriptor, err = resolver.resolve(fullMethod, raw)
		if err == nil {
			dyn := dynamic.NewMessage(descriptor)
			// now unmarshal again using the new generated message type
			err = proto.Unmarshal(raw, dyn)
			if err == nil {
				return dyn, nil
			}
		}
	}
	return nil, err
}
