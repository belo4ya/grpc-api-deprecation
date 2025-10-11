package apideprecation

import (
	"context"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type config struct {
	extraLabels LabelSet
	exemplar    ExemplarSet
	seedDesc    []protoreflect.MessageDescriptor
}

type LabelSet struct {
	Field []Label
	Enum  []Label
}

type ExemplarSet = LabelSet

type Label struct {
	Name  string
	Value LabelValueFunc
}

type LabelValueFunc func(ctx context.Context, msg proto.Message, fd protoreflect.FieldDescriptor) string

type Option func(*config)

func WithExtraLabels(extraLabels LabelSet) Option {
	return func(c *config) {
		c.extraLabels = extraLabels
	}
}

func WithExemplar(exemplar ExemplarSet) Option {
	return func(c *config) {
		c.exemplar = exemplar
	}
}

func WithMessageDescriptors(descriptors ...protoreflect.MessageDescriptor) Option {
	return func(c *config) {
		c.seedDesc = descriptors
	}
}
