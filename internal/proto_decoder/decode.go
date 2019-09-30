package proto_decoder

import (
	"context"
	"github.com/bradleyjkemp/grpc-tools/internal"
	"github.com/golang/protobuf/proto"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/metadata"
)

type MessageResolver interface {
	// takes an encoded message and finds a message descriptor for it
	// so it can be unmarshalled into an object
	resolveEncoded(ctx context.Context, fullMethod string, message *internal.Message, metadata metadata.MD) (*desc.MessageDescriptor, error)

	// takes a message object and finds a message descriptor for it
	// so it can be marshalled back into bytes
	resolveDecoded(ctx context.Context, fullMethod string, message *internal.Message, metadata metadata.MD) (*desc.MessageDescriptor, error)
}

type MessageDecoder interface {
	Decode(fullMethod string, message *internal.Message, metadata metadata.MD) (*dynamic.Message, error)
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

func (d *messageDecoder) Decode(fullMethod string, message *internal.Message, md metadata.MD) (*dynamic.Message, error) {
	var err error
	var descriptor *desc.MessageDescriptor
	for _, resolver := range d.resolvers {
		descriptor, err = resolver.resolveEncoded(context.Background(), fullMethod, message, md)
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
