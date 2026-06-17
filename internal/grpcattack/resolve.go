package grpcattack

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/bufbuild/protocompile"
	"google.golang.org/grpc"
	reflectpb "google.golang.org/grpc/reflection/grpc_reflection_v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

func ResolveMethod(cfg *Config, conn *grpc.ClientConn) (protoreflect.MethodDescriptor, error) {
	if cfg.ProtoFile != "" {
		return resolveFromProtoFile(cfg.ProtoFile, cfg.Service, cfg.Method)
	}
	return resolveFromReflection(conn, cfg.Service, cfg.Method)
}

func resolveFromProtoFile(protoFile, serviceName, methodName string) (protoreflect.MethodDescriptor, error) {
	compiler := protocompile.Compiler{
		Resolver: protocompile.WithStandardImports(&protocompile.SourceResolver{
			ImportPaths: []string{filepath.Dir(protoFile)},
		}),
	}

	files, err := compiler.Compile(context.Background(), filepath.Base(protoFile))
	if err != nil {
		return nil, fmt.Errorf("compile %q: %w", protoFile, err)
	}

	for _, fd := range files {
		svcs := fd.Services()
		for i := 0; i < svcs.Len(); i++ {
			svc := svcs.Get(i)
			if string(svc.FullName()) != serviceName {
				continue
			}
			mds := svc.Methods()
			for j := 0; j < mds.Len(); j++ {
				md := mds.Get(j)
				if string(md.Name()) == methodName {
					return md, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("method %q not found in service %q in %q", methodName, serviceName, protoFile)
}

func resolveFromReflection(conn *grpc.ClientConn, serviceName, methodName string) (protoreflect.MethodDescriptor, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, err := reflectpb.NewServerReflectionClient(conn).ServerReflectionInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("open reflection stream: %w", err)
	}
	defer stream.CloseSend()

	if err := stream.Send(&reflectpb.ServerReflectionRequest{
		MessageRequest: &reflectpb.ServerReflectionRequest_FileContainingSymbol{
			FileContainingSymbol: serviceName,
		},
	}); err != nil {
		return nil, fmt.Errorf("send reflection request: %w", err)
	}

	resp, err := stream.Recv()
	if err != nil {
		return nil, fmt.Errorf("recv reflection response: %w", err)
	}
	if errResp := resp.GetErrorResponse(); errResp != nil {
		return nil, fmt.Errorf("reflection error %d: %s (is reflection enabled on the server?)",
			errResp.ErrorCode, errResp.ErrorMessage)
	}

	fdr := resp.GetFileDescriptorResponse()
	if fdr == nil {
		return nil, fmt.Errorf("unexpected reflection response type")
	}

	fdset := &descriptorpb.FileDescriptorSet{}
	for _, raw := range fdr.GetFileDescriptorProto() {
		fdp := &descriptorpb.FileDescriptorProto{}
		if err := proto.Unmarshal(raw, fdp); err != nil {
			return nil, fmt.Errorf("unmarshal FileDescriptorProto: %w", err)
		}
		fdset.File = append(fdset.File, fdp)
	}

	registry, err := protodesc.NewFiles(fdset)
	if err != nil {
		return nil, fmt.Errorf("build file registry: %w", err)
	}

	return findMethod(registry, serviceName, methodName)
}

func findMethod(registry interface {
	RangeFiles(func(protoreflect.FileDescriptor) bool)
}, serviceName, methodName string) (protoreflect.MethodDescriptor, error) {
	var found protoreflect.MethodDescriptor
	registry.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		svcs := fd.Services()
		for i := 0; i < svcs.Len(); i++ {
			svc := svcs.Get(i)
			if string(svc.FullName()) != serviceName {
				continue
			}
			mds := svc.Methods()
			for j := 0; j < mds.Len(); j++ {
				md := mds.Get(j)
				if string(md.Name()) == methodName {
					found = md
					return false
				}
			}
		}
		return true
	})
	if found == nil {
		return nil, fmt.Errorf("method %q not found in service %q", methodName, serviceName)
	}
	return found, nil
}
