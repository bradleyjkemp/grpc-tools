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

type MessageEncoder interface {
	Encode(fullMethod string, message *internal.Message) ([]byte, error)
}

// Chain together a number of resolvers to decode incoming messages.
// Resolvers are in priority order, the first to return a nil error
// is used to decode the message. If no resolvers are successful,
// a default resolver is used that always returns empty.Empty
func NewEncoder(resolvers ...MessageResolver) *messageEncoder {
	return &messageEncoder{
		resolvers: append(resolvers),
		// TODO: include an unknown message encoder here
	}
}

func (d *messageEncoder) Encode(fullMethod string, message *internal.Message) ([]byte, error) {
	switch {
	case message.Message == nil && message.RawMessage != nil:
		return message.RawMessage, nil

	case message.Message != nil && message.RawMessage != nil:
		msgBytes, err := d.encodeFromHumanReadable(fullMethod, message)
		if err != nil {
			// TODO: log warning here
			return message.RawMessage, nil
		}
		return msgBytes, nil

	case message.Message != nil && message.RawMessage == nil:
		// Not possible to fall back to using the raw message so return directly
		return d.encodeFromHumanReadable(fullMethod, message)

	default:
		return nil, fmt.Errorf("no message available: both Message and RawMessage are nil")
	}
}

func (d *messageEncoder) encodeFromHumanReadable(fullMethod string, message *internal.Message) ([]byte, error) {
	if len(d.resolvers) == 0 {
		return nil, fmt.Errorf("no resolvers available")
	}

	var err error
	for _, resolver := range d.resolvers {
		var descriptor *desc.MessageDescriptor
		descriptor, err = resolver.resolveDecoded(fullMethod, message)
		if err != nil {
			continue
		}

		var jsonMarshalled []byte
		jsonMarshalled, err = json.Marshal(message.Message)
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
