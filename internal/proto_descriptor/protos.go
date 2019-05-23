package proto_descriptor

import (
	"fmt"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/protoparse"
	"os"
	"path/filepath"
)

func LoadProtoDescriptors(descriptorPaths ...string) (map[string]*desc.MethodDescriptor, error) {
	descriptors := []*desc.FileDescriptor{}
	for _, path := range descriptorPaths {
		descriptor, err := desc.LoadFileDescriptor(path)
		if err != nil {
			return nil, err
		}
		descriptors = append(descriptors, descriptor)
	}

	return convertDescriptorsToMap(descriptors), nil
}

// recursively walks through all files in the given directories and
// finds .proto files that contains service definitions
func LoadProtoDirectories(roots ...string) (map[string]*desc.MethodDescriptor, error) {
	var servicesFiles []*desc.FileDescriptor

	parser := protoparse.Parser{
		ImportPaths:      roots,
		InferImportPaths: true, // attempt to be clever
	}

	// scan all roots for .proto files containing gRPC service definitions
	for _, root := range roots {
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if filepath.Ext(path) == ".proto" {
				relpath, err := filepath.Rel(root, path)
				if err != nil {
					return err
				}
				descs, err := parser.ParseFilesButDoNotLink(path)
				if err != nil {
					// oh well we won't worry though
					fmt.Fprintf(os.Stderr, "Skipping %s due to parse error %s", path, err)
					return nil
				}
				if len(descs[0].Service) > 0 {
					// this file is interesting so
					fileDesc, err := parser.ParseFiles(relpath)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Skipping %s due to parse error %s", path, err)
						return nil
					}
					servicesFiles = append(servicesFiles, fileDesc[0])
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	if len(servicesFiles) == 0 {
		return nil, fmt.Errorf("no service definitions found")
	}

	return convertDescriptorsToMap(servicesFiles), nil
}

func convertDescriptorsToMap(descs []*desc.FileDescriptor) map[string]*desc.MethodDescriptor {
	methods := map[string]*desc.MethodDescriptor{}
	for _, desc := range descs {
		for _, service := range desc.GetServices() {
			for _, method := range service.GetMethods() {
				// have to convert fully qualified name com.service.Method
				// into gRPC "info.FullMethod" format /com.service/Method
				methodName := method.GetName()
				serviceName := method.GetParent().GetFullyQualifiedName()
				methods[fmt.Sprintf("/%s/%s", serviceName, methodName)] = method
			}
		}
	}

	return methods
}
