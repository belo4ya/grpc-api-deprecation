package apideprecation

import (
	"maps"
	"sync"
	"sync/atomic"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

type planCache map[protoreflect.MessageDescriptor]*evalPlan

func (c planCache) Clone() planCache {
	newCache := make(planCache, len(c)+1)
	maps.Copy(newCache, c)
	return newCache
}

type planBuilder struct {
	mu    sync.Mutex                // serializes cache writes
	cache atomic.Pointer[planCache] // copy-on-write cache
}

func newPlanBuilder() *planBuilder {
	b := &planBuilder{}
	cache := planCache{}
	b.cache.Store(&cache)
	return b
}

func (b *planBuilder) LoadOrBuild(md protoreflect.MessageDescriptor) *evalPlan {
	if plan, ok := (*b.cache.Load())[md]; ok {
		return plan
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	cache := *b.cache.Load()
	if plan, ok := cache[md]; ok { // TOCTOU
		return plan
	}
	newCache := cache.Clone()
	plan := b.build(md, newCache)
	b.cache.Store(&newCache)
	return plan
}

func (b *planBuilder) build(md protoreflect.MessageDescriptor, cache planCache) *evalPlan {
	if plan, ok := cache[md]; ok {
		return plan
	}
	plan := &evalPlan{}
	cache[md] = plan
	b.processFields(md, plan, cache)
	return plan
}

func (b *planBuilder) processFields(md protoreflect.MessageDescriptor, plan *evalPlan, cache planCache) {
	fields := md.Fields()
	for i := range fields.Len() {
		fd := fields.Get(i)

		if isFieldDeprecated(fd) {
			plan.Append(newFieldNode(fd))
			continue
		}

		if fd.IsMap() {
			mv := fd.MapValue()
			switch mv.Kind() {
			case protoreflect.MessageKind:
				if nested := b.build(mv.Message(), cache); len(nested.evaluators) != 0 {
					plan.Append(newMapNode(fd, nested))
				}
			case protoreflect.EnumKind:
				if deprecated := collectDeprecatedEnumValues(mv.Enum()); len(deprecated) != 0 {
					plan.Append(newMapNode(fd, newEnumNode(fd, deprecated)))
				}
			}
			continue
		}

		if fd.IsList() {
			switch fd.Kind() {
			case protoreflect.MessageKind:
				if nested := b.build(fd.Message(), cache); len(nested.evaluators) != 0 {
					plan.Append(newListNode(fd, nested))
				}
			case protoreflect.EnumKind:
				if deprecated := collectDeprecatedEnumValues(fd.Enum()); len(deprecated) != 0 {
					plan.Append(newListNode(fd, newEnumNode(fd, deprecated)))
				}
			}
			continue
		}

		switch fd.Kind() {
		case protoreflect.MessageKind:
			if nested := b.build(fd.Message(), cache); len(nested.evaluators) != 0 {
				plan.Append(newMessageNode(fd, nested))
			}
		case protoreflect.EnumKind:
			if deprecated := collectDeprecatedEnumValues(fd.Enum()); len(deprecated) != 0 {
				plan.Append(newEnumNode(fd, deprecated))
			}
		}
	}
}

func isFieldDeprecated(fd protoreflect.FieldDescriptor) bool {
	opts, ok := fd.Options().(*descriptorpb.FieldOptions)
	return ok && opts.GetDeprecated()
}

func collectDeprecatedEnumValues(ed protoreflect.EnumDescriptor) map[protoreflect.EnumNumber]protoreflect.Name {
	deprecated := map[protoreflect.EnumNumber]protoreflect.Name{}
	enums := ed.Values()
	for i := range enums.Len() {
		if evd := enums.Get(i); isEnumValueDeprecated(evd) {
			deprecated[evd.Number()] = evd.Name()
		}
	}
	return deprecated
}

func isEnumValueDeprecated(evd protoreflect.EnumValueDescriptor) bool {
	opts, ok := evd.Options().(*descriptorpb.EnumValueOptions)
	return ok && opts.GetDeprecated()
}
