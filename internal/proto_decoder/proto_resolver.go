package proto_decoder

import (
	"fmt"
	"github.com/bradleyjkemp/grpc-tools/internal"
	"github.com/bradleyjkemp/grpc-tools/internal/proto_descriptor"
	"github.com/jhump/protoreflect/desc"
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
