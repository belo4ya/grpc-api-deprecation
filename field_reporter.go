package apideprecation

import (
	"maps"
	"sync"
	"sync/atomic"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

type fieldReporter struct {
	mu    sync.Mutex                // serializes cache writes
	cache atomic.Pointer[planCache] // copy-on-write cache
}

func newFieldReporter(seedDesc []protoreflect.MessageDescriptor) *fieldReporter {
	r := &fieldReporter{}
	cache := make(planCache, len(seedDesc))
	for _, desc := range seedDesc {
		r.buildPlan(desc, cache)
	}
	r.cache.Store(&cache)
	return r
}

func (r *fieldReporter) Report(
	msg protoreflect.Message,
	meta CallMeta,
	onDeprecatedField onDeprecatedFieldFunc,
	onDeprecatedEnum onDeprecatedEnumFunc,
) {
	plan := r.loadOrBuildPlan(msg.Descriptor())
	plan.EvalMessage(msg, meta, onDeprecatedField, onDeprecatedEnum)
}

type planCache map[protoreflect.MessageDescriptor]*evalPlan

func (c planCache) Clone() planCache {
	newCache := make(planCache, len(c)+1)
	maps.Copy(newCache, c)
	return newCache
}

func (r *fieldReporter) loadOrBuildPlan(md protoreflect.MessageDescriptor) *evalPlan {
	if plan, ok := (*r.cache.Load())[md]; ok {
		return plan
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	cache := *r.cache.Load()
	if plan, ok := cache[md]; ok { // TOCTOU
		return plan
	}
	newCache := cache.Clone()
	plan := r.buildPlan(md, newCache)
	r.cache.Store(&newCache)
	return plan
}

func (r *fieldReporter) buildPlan(md protoreflect.MessageDescriptor, cache planCache) *evalPlan {
	if plan, ok := cache[md]; ok {
		return plan
	}
	plan := &evalPlan{}
	cache[md] = plan
	r.processFields(md, plan, cache)
	return plan
}

func (r *fieldReporter) processFields(md protoreflect.MessageDescriptor, plan *evalPlan, cache planCache) {
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
				if nested := r.buildPlan(mv.Message(), cache); len(nested.evaluators) != 0 {
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
				if nested := r.buildPlan(fd.Message(), cache); len(nested.evaluators) != 0 {
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
			if nested := r.buildPlan(fd.Message(), cache); len(nested.evaluators) != 0 {
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
