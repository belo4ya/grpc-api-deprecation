package apideprecation

import (
	"sync"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

type methodReporter struct {
	cache sync.Map
}

func newMethodReporter() *methodReporter {
	return &methodReporter{cache: sync.Map{}}
}

func (r *methodReporter) Report(meta callMeta, onDeprecated func()) bool {
	if r.isDeprecated(meta) {
		onDeprecated()
		return true
	}
	return false
}

func (r *methodReporter) isDeprecated(meta callMeta) bool {
	if v, ok := r.cache.Load(meta.FullMethod); ok {
		return v.(bool)
	}
	deprecated := r.lookupDeprecated(meta)
	r.cache.Store(meta.FullMethod, deprecated)
	return deprecated
}

func (r *methodReporter) lookupDeprecated(meta callMeta) bool {
	desc, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(meta.Service))
	if err != nil {
		return false
	}
	service, ok := desc.(protoreflect.ServiceDescriptor)
	if !ok {
		return false
	}
	method := service.Methods().ByName(protoreflect.Name(meta.Method))
	if method == nil {
		return false
	}
	opts, ok := method.Options().(*descriptorpb.MethodOptions)
	return ok && opts.GetDeprecated()
}
