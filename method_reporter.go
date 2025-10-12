package apideprecation

import (
	"sync"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

type methodReporter struct {
	cache sync.Map // string[bool] (fullMethod -> isDeprecated)
}

func newMethodReporter(seedDesc []protoreflect.ServiceDescriptor) *methodReporter {
	r := &methodReporter{}
	for _, sd := range seedDesc {
		methods := sd.Methods()
		for i := 0; i < methods.Len(); i++ {
			r.IsDeprecated(string(methods.Get(i).FullName()))
		}
	}
	return r
}

func (r *methodReporter) IsDeprecated(fullMethod string) bool {
	if v, ok := r.cache.Load(fullMethod); ok {
		return v.(bool)
	}
	deprecated := r.isDeprecated(fullMethod)
	r.cache.Store(fullMethod, deprecated)
	return deprecated
}

func (r *methodReporter) isDeprecated(fullMethod string) bool {
	desc, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(fullMethod))
	if err != nil {
		return false
	}
	md, ok := desc.(protoreflect.MethodDescriptor)
	if !ok {
		return false
	}
	return r.resolveMethod(md)
}

func (r *methodReporter) resolveMethod(md protoreflect.MethodDescriptor) bool {
	if isMethodDeprecated(md) {
		return true
	}
	sd, ok := md.Parent().(protoreflect.ServiceDescriptor)
	return ok && isServiceDeprecated(sd)
}

func isMethodDeprecated(md protoreflect.MethodDescriptor) bool {
	opts, ok := md.Options().(*descriptorpb.MethodOptions)
	return ok && opts.GetDeprecated()
}

func isServiceDeprecated(sd protoreflect.ServiceDescriptor) bool {
	opts, ok := sd.Options().(*descriptorpb.ServiceOptions)
	return ok && opts.GetDeprecated()
}
