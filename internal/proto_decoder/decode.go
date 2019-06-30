package proto_decoder

import (
	"github.com/bradleyjkemp/grpc-tools/internal"
	"github.com/golang/protobuf/proto"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
)

type MessageResolver interface {
	resolve(fullMethod string, direction internal.MessageOrigin, raw []byte) (*desc.MessageDescriptor, error)
}

type MessageDecoder interface {
	Decode(fullMethod string, direction internal.MessageOrigin, raw []byte) (*dynamic.Message, error)
}

type messageDecoder struct {
	resolvers []MessageResolver
}

// Chain together a number of resolvers to decode incoming messages.
// Resolvers are in priority order, the first to return a nil error
// is used to decode the message.
func NewDecoder(resolvers ...MessageResolver) *messageDecoder {
	return &messageDecoder{
		resolvers: resolvers,
	}
}

func (d *messageDecoder) Decode(fullMethod string, direction internal.MessageOrigin, raw []byte) (*dynamic.Message, error) {
	var err error
	for _, resolver := range d.resolvers {
		var descriptor *desc.MessageDescriptor
		descriptor, err = resolver.resolve(fullMethod, direction, raw)
		if err != nil {
			continue
		}
		dyn := dynamic.NewMessage(descriptor)
		// now unmarshal again using the new generated message type
		err = proto.Unmarshal(raw, dyn)
		if err == nil {
			return dyn, nil
		}
	}
	return nil, err
}
