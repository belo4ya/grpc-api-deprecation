package apideprecation

import (
	"strings"
	"sync"

	"google.golang.org/protobuf/reflect/protoreflect"
)

func renderFieldPathPart(fd protoreflect.FieldDescriptor) string {
	name := string(fd.Name())
	if fd.IsList() {
		return name + "[]"
	}
	if fd.IsMap() {
		return name + "{}"
	}
	return name
}

var fieldPathPool = sync.Pool{
	New: func() any { return &fieldPath{parts: make([]string, 0, 8)} },
}

func newFieldPath() *fieldPath {
	return fieldPathPool.Get().(*fieldPath)
}

type fieldPath struct {
	parts []string
}

func (p *fieldPath) Release() {
	p.parts = p.parts[:0]
	fieldPathPool.Put(p)
}

func (p *fieldPath) Push(part string) {
	p.parts = append(p.parts, part)
}

func (p *fieldPath) Pop() {
	p.parts = p.parts[:len(p.parts)-1]
}

func (p *fieldPath) Render() string {
	size := len(p.parts) - 1 // +dots, e.g. a.b.c
	cut := false
	for i, part := range p.parts {
		size += len(part)
		if i == len(p.parts)-1 {
			if v := part[len(part)-1]; v == ']' || v == '}' {
				cut = true
				size -= 2
			}
		}
	}

	var sb strings.Builder
	sb.Grow(size)
	for i, part := range p.parts {
		if i > 0 {
			sb.WriteByte('.')
		}
		if i == len(p.parts)-1 && cut {
			sb.WriteString(part[:len(part)-2])
		} else {
			sb.WriteString(part)
		}
	}
	return sb.String()
}
