package proto_decoder

import (
	"encoding/json"
	"fmt"
	"github.com/bradleyjkemp/grpc-tools/internal"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
)

type messageEncoder struct {
	resolvers []MessageResolver
}

// Chain together a number of resolvers to decode incoming messages.
// Resolvers are in priority order, the first to return a nil error
// is used to decode the message.
func NewEncoder(resolvers ...MessageResolver) *messageEncoder {
	return &messageEncoder{
		resolvers: resolvers,
	}
}

func (d *messageEncoder) Encode(fullMethod string, direction internal.MessageOrigin, message interface{}) ([]byte, error) {
	if len(d.resolvers) == 0 {
		return nil, fmt.Errorf("no resolvers available")
	}

	var err error
	for _, resolver := range d.resolvers {
		var descriptor *desc.MessageDescriptor
		descriptor, err = resolver.resolveDecoded(fullMethod, direction, message)
		if err != nil {
			continue
		}

		jsonMarshalled, err := json.Marshal(message)
		if err != nil {
			continue
		}

		// now unmarshal again using the new generated message type
		dyn := dynamic.NewMessage(descriptor)
		err = jsonpb.UnmarshalString(string(jsonMarshalled), dyn)
		if err != nil {
			continue
		}
		var b []byte
		b, err = proto.Marshal(dyn)
		if err == nil {
			return b, nil
		}
	}
	return nil, err
}
