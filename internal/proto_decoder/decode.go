package proto_decoder

import (
	"errors"
	"github.com/bradleyjkemp/grpc-tools/internal"
	"github.com/golang/protobuf/proto"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/sirupsen/logrus"
)

type MessageResolver interface {
	// takes an encoded message and finds a message descriptor for it
	// so it can be unmarshalled into an object
	resolveEncoded(fullMethod string, message *internal.Message) (*desc.MessageDescriptor, error)

	// takes a message object and finds a message descriptor for it
	// so it can be marshalled back into bytes
	resolveDecoded(fullMethod string, message *internal.Message) (*desc.MessageDescriptor, error)
}

type MessageDecoder interface {
	Decode(fullMethod string, message *internal.Message) (*dynamic.Message, error)
}

type messageDecoder struct {
	logger       logrus.FieldLogger
	resolvers    []MessageResolver
	unknownField unknownFieldResolver
}

// Chain together a number of resolvers to decode incoming messages.
// Resolvers are in priority order, the first to return a nil error
// is used to decode the message.
func NewDecoder(logger logrus.FieldLogger, resolvers ...MessageResolver) *messageDecoder {
	return &messageDecoder{
		logger:       logger.WithField("", "proto_decoder"),
		resolvers:    append(resolvers, emptyResolver{}),
		unknownField: unknownFieldResolver{},
	}
}

func (d *messageDecoder) Decode(fullMethod string, message *internal.Message) (*dynamic.Message, error) {
	if len(d.resolvers) == 0 {
		return nil, errors.New("no MessageResolvers available")
	}

	var err error
	var descriptor *desc.MessageDescriptor
	for _, resolver := range d.resolvers {
		descriptor, err = resolver.resolveEncoded(fullMethod, message)
		if err == nil {
			break
		}
	}
	if err != nil {
		return nil, err
	}

	// check for any unknown fields and add them to the descriptor
	enrichedDescriptor, err := d.unknownField.enrichDecodeDescriptor(descriptor, message)
	if err == nil {
		descriptor = enrichedDescriptor
	} else {
		d.logger.WithError(err).Warn("Failed to search for unknown fields in message")
	}

	// now unmarshal using the resolved message type
	dyn := dynamic.NewMessage(descriptor)
	err = proto.Unmarshal(message.RawMessage, dyn)
	if err == nil {
		return dyn, nil
	}
	return nil, err
}
