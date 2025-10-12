package apideprecation

import (
	"strings"
	"sync"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

type methodReporter struct {
	cache sync.Map // fullMethod -> methodCacheEntry
}

func newMethodReporter(seedDesc []protoreflect.ServiceDescriptor) *methodReporter {
	r := &methodReporter{}
	for _, sd := range seedDesc {
		service := string(sd.FullName())
		methods := sd.Methods()
		for i := 0; i < methods.Len(); i++ {
			method := string(methods.Get(i).Name())
			r.getOrResolve("/" + service + "/" + method)
		}
	}
	return r
}

func (r *methodReporter) Report(fullMethod string, onDeprecated func(md protoreflect.MethodDescriptor)) bool {
	if entry := r.getOrResolve(fullMethod); entry.deprecated {
		onDeprecated(entry.md)
		return true
	}
	return false
}

type methodCacheEntry struct {
	deprecated bool
	md         protoreflect.MethodDescriptor
}

func (r *methodReporter) getOrResolve(fullMethod string) methodCacheEntry {
	if v, ok := r.cache.Load(fullMethod); ok {
		return v.(methodCacheEntry)
	}
	entry := r.resolveDescriptor(fullMethod)
	r.cache.Store(fullMethod, entry)
	return entry
}

func (r *methodReporter) resolveDescriptor(fullMethod string) methodCacheEntry {
	desc, err := protoregistry.GlobalFiles.FindDescriptorByName(r.fullMethodToName(fullMethod))
	if err != nil {
		return methodCacheEntry{deprecated: false}
	}
	md, ok := desc.(protoreflect.MethodDescriptor)
	if !ok {
		return methodCacheEntry{deprecated: false}
	}
	if !r.isMethodOrServiceDeprecated(md) {
		return methodCacheEntry{deprecated: false}
	}
	return methodCacheEntry{deprecated: true, md: md}
}

func (r *methodReporter) fullMethodToName(fullMethod string) protoreflect.FullName {
	i := strings.LastIndexByte(fullMethod, '/')
	return protoreflect.FullName(fullMethod[1:i] + "." + fullMethod[i+1:])
}

func (r *methodReporter) isMethodOrServiceDeprecated(md protoreflect.MethodDescriptor) bool {
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
