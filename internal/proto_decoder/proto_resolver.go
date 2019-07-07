package proto_decoder

import (
	"fmt"
	"github.com/bradleyjkemp/grpc-tools/internal"
	"github.com/bradleyjkemp/grpc-tools/internal/proto_descriptor"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/builder"
	"strings"
)

type descriptorResolver struct {
	methodDescriptors map[string]*desc.MethodDescriptor
}

func (d *descriptorResolver) resolveEncoded(fullMethod string, message *internal.Message) (*desc.MessageDescriptor, error) {
	return d.resolve(fullMethod, message.MessageOrigin)
}

func (d *descriptorResolver) resolveDecoded(fullMethod string, message *internal.Message) (*desc.MessageDescriptor, error) {
	return d.resolve(fullMethod, message.MessageOrigin)
}

func (d *descriptorResolver) resolve(fullMethod string, direction internal.MessageOrigin) (*desc.MessageDescriptor, error) {
	if descriptor, ok := d.methodDescriptors[fullMethod]; ok {
		switch direction {
		case internal.ClientMessage:
			return descriptor.GetInputType(), nil
		case internal.ServerMessage:
			return descriptor.GetOutputType(), nil
		}
	}
	return nil, fmt.Errorf("method not known")
}

func NewFileResolver(protoFileRoots ...string) (*descriptorResolver, error) {
	descs, err := proto_descriptor.LoadProtoDirectories(protoFileRoots...)
	if err != nil {
		return nil, err
	}

	return &descriptorResolver{
		descs,
	}, nil
}

func NewDescriptorResolver(protoFileDescriptors ...string) (*descriptorResolver, error) {
	descs, err := proto_descriptor.LoadProtoDescriptors(protoFileDescriptors...)
	if err != nil {
		return nil, err
	}

	return &descriptorResolver{
		descs,
	}, nil
}

var messageName = strings.NewReplacer(
	"/", "_",
	".", "_",
)

type emptyResolver struct{}

func (e emptyResolver) resolveEncoded(fullMethod string, message *internal.Message) (*desc.MessageDescriptor, error) {
	d, err := desc.LoadMessageDescriptorForMessage(&empty.Empty{})
	if err != nil {
		return nil, err
	}
	mb, err := builder.FromMessage(d)
	if err != nil {
		return nil, err
	}
	mb.SetName(fmt.Sprintf("%s_%s", messageName.Replace(fullMethod), message.MessageOrigin))
	return mb.Build()
}

func (e emptyResolver) resolveDecoded(fullMethod string, message *internal.Message) (*desc.MessageDescriptor, error) {
	return desc.LoadMessageDescriptorForMessage(&empty.Empty{})
}
