package proto_decoder

import (
	"fmt"
	"github.com/bradleyjkemp/grpc-tools/internal"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/builder"
	"github.com/jhump/protoreflect/dynamic"
	"regexp"
)

// When we don't have an actual proto message descriptor, this takes a best effort
// approach to generating one. It's definitely not perfect but is more useful than nothing.

type unknownMessageResolver struct {
	messageCounter int64 // must only be accessed atomically
}

func NewUnknownResolver() *unknownMessageResolver {
	return &unknownMessageResolver{
		messageCounter: 0,
	}
}

// This takes a message descriptor and enriches it to add any unknown fields present.
// This means that all unknown fields will show up in the dump.
func (u *unknownMessageResolver) enrichDecodeDescriptor(resolved *desc.MessageDescriptor, message *internal.Message) (*desc.MessageDescriptor, error) {
	decoded := dynamic.NewMessage(resolved)
	err := proto.Unmarshal(message.RawMessage, decoded)
	if err != nil {
		return nil, err
	}
	descriptor, err := builder.FromMessage(resolved)
	if err != nil {
		return nil, err
	}
	err = u.enrichMessage(descriptor, decoded)
	if err != nil {
		return nil, err
	}
	return descriptor.Build()
}

func (u *unknownMessageResolver) enrichMessage(descriptor *builder.MessageBuilder, message *dynamic.Message) error {
	for _, fieldNum := range message.GetUnknownFields() {
		unknownFieldContents := message.GetUnknownField(fieldNum)
		fieldType, err := u.detectFieldType(unknownFieldContents)
		if err != nil {
			return err
		}
		field := builder.NewField(fmt.Sprintf("_%d", fieldNum), fieldType)
		field.SetNumber(fieldNum)
		if len(unknownFieldContents) > 1 {
			field.SetRepeated()
		}
		if descriptor.TryAddField(field) != nil {
			return err
		}
	}

	// recurse into the known fields to check for nested unknown fields
	for _, fieldDescriptor := range message.GetKnownFields() {
		if fieldDescriptor.GetMessageType() != nil {
			nestedMessage, ok := message.GetField(fieldDescriptor).(proto.Message)
			if !ok {
				return fmt.Errorf("error: nested message was not of type proto.Message")
			}
			nestedMessageDescriptor, err := builder.FromMessage(fieldDescriptor.GetMessageType())
			if err != nil {
				return err
			}
			dynamicNestedMessage := dynamic.NewMessage(fieldDescriptor.GetMessageType())
			err = dynamicNestedMessage.MergeFrom(nestedMessage)
			if err != nil {
				return err
			}
			err = u.enrichMessage(nestedMessageDescriptor, dynamicNestedMessage)
			if err != nil {
				return err
			}
			fieldDescriptorBuilder, err := builder.FromField(fieldDescriptor)
			if err != nil {
				return err
			}
			fieldDescriptorBuilder.SetType(builder.FieldTypeMessage(nestedMessageDescriptor))
			descriptor.RemoveField(fieldDescriptor.GetName()).AddField(fieldDescriptorBuilder)
		}
	}

	return nil
}

func (u *unknownMessageResolver) resolveDecoded(fullMethod string, message *internal.Message) (*desc.MessageDescriptor, error) {
	return nil, fmt.Errorf("unimplemented")
}

var (
	asciiPattern = regexp.MustCompile(`^[ -~]*$`)
)

func (u *unknownMessageResolver) detectFieldType(fields []dynamic.UnknownField) (*builder.FieldType, error) {
	field := fields[0]
	switch field.Encoding {
	// TODO: handle all wire types
	case proto.WireBytes:
		if asciiPattern.Match(field.Contents) {
			// highly unlikely that an entirely ASCII string is actually an embedded proto message
			// TODO: make this heuristic cleverer
			return builder.FieldTypeString(), nil
		}
		// embedded messages are encoded on the wire as strings
		// so try to decode this string as a message
		dyn, err := dynamic.AsDynamicMessage(&empty.Empty{})
		if err != nil {
			panic(err)
		}
		err = proto.Unmarshal(field.Contents, dyn)
		if err != nil {
			// looks like it wasn't a valid proto message
			return builder.FieldTypeString(), nil
		}
		// TODO: check that the unmarshalled message doesn't have any illegal field numbers

		// probably is an embedded message
		descriptor, _ := builder.FromMessage(dyn.GetMessageDescriptor())
		err = u.enrichMessage(descriptor, dyn)
		if err != nil {
			return nil, err
		}
		return builder.FieldTypeMessage(descriptor), nil

	default:
		// Fixed precision number
		return builder.FieldTypeFixed64(), nil
	}
}
