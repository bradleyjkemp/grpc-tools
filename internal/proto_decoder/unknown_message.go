package proto_decoder

import (
	"fmt"
	"github.com/bradleyjkemp/grpc-tools/internal"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/builder"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/pkg/errors"
	"regexp"
)

// When we don't have an actual proto message descriptor, this takes a best effort
// approach to generating one. It's definitely not perfect but is more useful than nothing.

type unknownFieldResolver struct{}

// This takes a message descriptor and enriches it to add any unknown fields present.
// This means that all unknown fields will show up in the dump.
func (u *unknownFieldResolver) enrichDecodeDescriptor(resolved *desc.MessageDescriptor, message *internal.Message) (*desc.MessageDescriptor, error) {
	decoded := dynamic.NewMessage(resolved)
	err := proto.Unmarshal(message.RawMessage, decoded)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal message")
	}
	descriptor, err := builder.FromMessage(resolved)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create builder for message")
	}
	err = u.enrichMessage(descriptor, decoded)
	if err != nil {
		return nil, errors.Wrap(err, "failed to enrich decode descriptor")
	}

	decodeDescriptor, err := descriptor.Build()
	if err != nil {
		return nil, errors.Wrap(err, "failed to build enriched decode descriptor")
	}
	return decodeDescriptor, nil
}

func (u *unknownFieldResolver) enrichMessage(descriptor *builder.MessageBuilder, message *dynamic.Message) error {
	for _, fieldNum := range message.GetUnknownFields() {
		generatedFieldName := fmt.Sprintf("%s_%d", descriptor.GetName(), fieldNum)
		unknownFieldContents := message.GetUnknownField(fieldNum)
		fieldType, err := u.detectUnknownFieldType(descriptor.GetFile(), generatedFieldName, unknownFieldContents)
		if err != nil {
			return errors.Wrap(err, "failed to detect field type")
		}

		field := builder.NewField(generatedFieldName, fieldType)
		if err := field.TrySetNumber(fieldNum); err != nil {
			return errors.Wrap(err, "failed to set field number")
		}
		field.SetJsonName(fmt.Sprintf("%d", fieldNum))
		if len(unknownFieldContents) > 1 {
			field.SetRepeated()
		}
		if err = descriptor.TryAddField(field); err != nil {
			return errors.Wrap(err, "failed to add field")
		}
	}

	// recurse into the known fields to check for nested unknown fields
	for _, fieldDescriptor := range message.GetKnownFields() {
		if fieldDescriptor.GetMessageType() == nil {
			// either this is a basic type
			// or a map which is not yet supported
			// TODO: support maps
			continue
		}

		var nestedMessage proto.Message
		switch field := message.GetField(fieldDescriptor).(type) {
		case proto.Message:
			nestedMessage = field

		// Repeated field: fieldDescriptor.IsRepeated() == true
		case []interface{}:
			if len(field) == 0 {
				// TODO: is this case possible?
				// Field has no items to analyse
				continue
			}
			var ok bool
			// TODO: should iterate over all the repeated messages and merge the information
			nestedMessage, ok = field[0].(proto.Message)
			if !ok {
				return fmt.Errorf("unknown: repeated field is not of type proto.Message")
			}

		default:
			return fmt.Errorf("unknown nested field type %T", field)
		}

		nestedMessageDescriptor, err := builder.FromMessage(fieldDescriptor.GetMessageType())
		if err != nil {
			return errors.Wrap(err, "failed to create builder")
		}

		dynamicNestedMessage, err := dynamic.AsDynamicMessage(nestedMessage)
		if err != nil {
			return errors.Wrap(err, "failed to convert nested message to dynamic")
		}

		err = u.enrichMessage(nestedMessageDescriptor, dynamicNestedMessage)
		if err != nil {
			return errors.Wrapf(err, "failed to search nested field %s", fieldDescriptor.GetName())
		}
		fieldDescriptorBuilder, err := builder.FromField(fieldDescriptor)
		if err != nil {
			return errors.Wrapf(err, "failed to create builder for field %s", fieldDescriptor.GetName())
		}
		fieldDescriptorBuilder.SetType(builder.FieldTypeMessage(nestedMessageDescriptor))
		descriptor.RemoveField(fieldDescriptor.GetName()).AddField(fieldDescriptorBuilder)
	}

	return nil
}

var (
	asciiPattern = regexp.MustCompile(`^[ -~]*$`)
)

func (u *unknownFieldResolver) detectUnknownFieldType(file *builder.FileBuilder, fieldName string, fields []dynamic.UnknownField) (*builder.FieldType, error) {
	field := fields[0]
	switch field.Encoding {
	// Used for: int32, int64, uint32, uint64, sint32, sint64, bool, enum
	case proto.WireVarint:
		return builder.FieldTypeInt64(), nil

	// Used for: fixed32, sfixed32, float
	case proto.WireFixed32:
		return builder.FieldTypeFloat(), nil

	// Used for: fixed64, sfixed64, double
	case proto.WireFixed64:
		return builder.FieldTypeDouble(), nil

	// Used for: string, bytes, embedded messages, packed repeated fields
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
		descriptor := builder.NewMessage(fieldName)
		err = u.enrichMessage(descriptor, dyn)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to detect unknown field %s", fieldName)
		}
		file.AddMessage(descriptor)
		return builder.FieldTypeMessage(descriptor), nil

	default:
		return nil, errors.Errorf("Unsupported wire type id %v", field.Encoding)
	}
}
