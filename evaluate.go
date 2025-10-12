package apideprecation

import (
	"google.golang.org/protobuf/reflect/protoreflect"
)

type evaluator interface {
	Eval(evalCtx evalContext, msg protoreflect.Message, val protoreflect.Value)
}

type evalContext struct {
	onDeprecatedField    onDeprecatedFieldFunc
	onDeprecatedEnum     onDeprecatedEnumFunc
	fieldPath            *fieldPath
	typ, service, method string
}

type (
	onDeprecatedFieldFunc func(fd protoreflect.FieldDescriptor, fieldFullName, fieldPresence string)
	onDeprecatedEnumFunc  func(fd protoreflect.FieldDescriptor, fieldFullName, enumValue string, enumNumber int)
)

type evalPlan struct {
	evaluators []evaluator
}

func (p *evalPlan) Append(eval evaluator) {
	p.evaluators = append(p.evaluators, eval)
}

func (p *evalPlan) EvalMessage(
	msg protoreflect.Message,
	meta callMeta,
	onDeprecatedField onDeprecatedFieldFunc,
	onDeprecatedEnum onDeprecatedEnumFunc,
) {
	fp := newFieldPath()
	defer fp.Release()
	p.Eval(evalContext{
		onDeprecatedField: onDeprecatedField,
		onDeprecatedEnum:  onDeprecatedEnum,
		fieldPath:         fp,
		typ:               meta.Type,
		service:           meta.Service,
		method:            meta.Method,
	}, msg, protoreflect.Value{})
}

func (p *evalPlan) Eval(evalCtx evalContext, msg protoreflect.Message, val protoreflect.Value) {
	for _, eval := range p.evaluators {
		eval.Eval(evalCtx, msg, val)
	}
}

// fieldNode evaluates a terminal (leaf) field that is marked as deprecated.
type fieldNode struct {
	fd       protoreflect.FieldDescriptor
	pathPart string
	presence string
}

func newFieldNode(fd protoreflect.FieldDescriptor) *fieldNode {
	return &fieldNode{
		fd:       fd,
		pathPart: renderFieldPathPart(fd),
		presence: presenceKind(fd),
	}
}

func (n *fieldNode) Eval(evalCtx evalContext, msg protoreflect.Message, _ protoreflect.Value) {
	if !msg.Has(n.fd) {
		return
	}
	evalCtx.fieldPath.Push(n.pathPart)
	evalCtx.onDeprecatedField(n.fd, evalCtx.fieldPath.Render(), n.presence)
	evalCtx.fieldPath.Pop()
}

func presenceKind(fd protoreflect.FieldDescriptor) string {
	if fd.HasPresence() {
		return "explicit"
	}
	return "implicit"
}

// enumNode evaluates a terminal (leaf) field or collection item that contains deprecated Enum value.
type enumNode struct {
	fd            protoreflect.FieldDescriptor
	deprecated    map[protoreflect.EnumNumber]protoreflect.Name
	fieldPathPart string
}

func newEnumNode(fd protoreflect.FieldDescriptor, deprecated map[protoreflect.EnumNumber]protoreflect.Name) *enumNode {
	return &enumNode{fd: fd, deprecated: deprecated, fieldPathPart: renderFieldPathPart(fd)}
}

func (n *enumNode) Eval(evalCtx evalContext, msg protoreflect.Message, val protoreflect.Value) {
	if val.IsValid() { // as collection item of listNode, mapNode nested.Eval()
		enum := val.Enum()
		if name, ok := n.deprecated[enum]; ok {
			evalCtx.onDeprecatedEnum(n.fd, evalCtx.fieldPath.Render(), string(name), int(enum))
		}
		return
	}

	// as message field
	if !msg.Has(n.fd) {
		return
	}
	enum := msg.Get(n.fd).Enum()
	if name, ok := n.deprecated[enum]; ok {
		evalCtx.fieldPath.Push(n.fieldPathPart)
		evalCtx.onDeprecatedEnum(n.fd, evalCtx.fieldPath.Render(), string(name), int(enum))
		evalCtx.fieldPath.Pop()
	}
}

// messageNode evaluates a nested Message field (never a leaf).
type messageNode struct {
	fd            protoreflect.FieldDescriptor
	nested        evaluator
	fieldPathPart string
}

func newMessageNode(fd protoreflect.FieldDescriptor, nested evaluator) *messageNode {
	return &messageNode{
		fd:            fd,
		nested:        nested,
		fieldPathPart: renderFieldPathPart(fd),
	}
}

func (n *messageNode) Eval(evalCtx evalContext, msg protoreflect.Message, _ protoreflect.Value) {
	if !msg.Has(n.fd) {
		return
	}
	evalCtx.fieldPath.Push(n.fieldPathPart)
	n.nested.Eval(evalCtx, msg.Get(n.fd).Message(), protoreflect.Value{})
	evalCtx.fieldPath.Pop()
}

// maxItemsPerCollection limits elements processed in repeated/map fields.
const maxItemsPerCollection = 50

// listNode evaluates a `repeated Message|Enum` field (never a leaf).
type listNode struct {
	fd            protoreflect.FieldDescriptor
	nested        evaluator
	fieldPathPart string
	evalItemValue func(evalCtx evalContext, val protoreflect.Value)
}

func newListNode(fd protoreflect.FieldDescriptor, nested evaluator) *listNode {
	n := &listNode{
		fd:            fd,
		nested:        nested,
		fieldPathPart: renderFieldPathPart(fd),
		evalItemValue: nil,
	}

	switch fd.Kind() {
	case protoreflect.MessageKind:
		n.evalItemValue = func(evalCtx evalContext, val protoreflect.Value) {
			n.nested.Eval(evalCtx, val.Message(), protoreflect.Value{})
		}
	case protoreflect.EnumKind:
		n.evalItemValue = func(evalCtx evalContext, val protoreflect.Value) {
			n.nested.Eval(evalCtx, nil, val)
		}
	}

	return n
}

func (n *listNode) Eval(evalCtx evalContext, msg protoreflect.Message, _ protoreflect.Value) {
	if !msg.Has(n.fd) {
		return
	}
	evalCtx.fieldPath.Push(n.fieldPathPart)
	list := msg.Get(n.fd).List()
	for i := range list.Len() {
		if i >= maxItemsPerCollection {
			hitMaxItemsPerCollectionInc(evalCtx.typ, evalCtx.service, evalCtx.method, evalCtx.fieldPath.Render(), "repeated", maxItemsPerCollection)
			break
		}
		n.evalItemValue(evalCtx, list.Get(i))
	}
	evalCtx.fieldPath.Pop()
}

// mapNode evaluates a `map<*, Message|Enum>` field (never a leaf).
type mapNode struct {
	fd            protoreflect.FieldDescriptor
	nested        evaluator
	fieldPathPart string
	evalItemValue func(evalCtx evalContext, val protoreflect.Value)
}

func newMapNode(fd protoreflect.FieldDescriptor, nested evaluator) *mapNode {
	n := &mapNode{
		fd:            fd,
		nested:        nested,
		fieldPathPart: renderFieldPathPart(fd),
		evalItemValue: nil,
	}

	switch fd.MapValue().Kind() {
	case protoreflect.MessageKind:
		n.evalItemValue = func(evalCtx evalContext, val protoreflect.Value) {
			n.nested.Eval(evalCtx, val.Message(), protoreflect.Value{})
		}
	case protoreflect.EnumKind:
		n.evalItemValue = func(evalCtx evalContext, val protoreflect.Value) {
			n.nested.Eval(evalCtx, nil, val)
		}
	}

	return n
}

func (n *mapNode) Eval(evalCtx evalContext, msg protoreflect.Message, _ protoreflect.Value) {
	if !msg.Has(n.fd) {
		return
	}
	evalCtx.fieldPath.Push(n.fieldPathPart)
	cnt := 0
	msg.Get(n.fd).Map().Range(func(_ protoreflect.MapKey, v protoreflect.Value) bool {
		if cnt >= maxItemsPerCollection {
			hitMaxItemsPerCollectionInc(evalCtx.typ, evalCtx.service, evalCtx.method, evalCtx.fieldPath.Render(), "map", maxItemsPerCollection)
			return false
		}
		n.evalItemValue(evalCtx, v)
		cnt++
		return true
	})
	evalCtx.fieldPath.Pop()
}
