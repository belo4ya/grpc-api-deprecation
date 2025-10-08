package apideprecation

import (
	"context"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type config struct {
	extraLabels ExtraLabels
	exemplars   Exemplars
	seedDesc    []protoreflect.MessageDescriptor
}

type ExtraLabels struct {
	Field map[string]labelValueFunc
	Enum  map[string]labelValueFunc
}

type Exemplars ExtraLabels

type labelValueFunc func(ctx context.Context, msg proto.Message, fd protoreflect.FieldDescriptor) string

type Option func(*config)

func WithExtraLabels(extraLabels ExtraLabels) Option {
	return func(c *config) {
		c.extraLabels = extraLabels
	}
}

func WithExemplars(exemplars Exemplars) Option {
	return func(c *config) {
		c.exemplars = exemplars
	}
}

func WithMessageDescriptors(descriptors ...protoreflect.MessageDescriptor) Option {
	return func(c *config) {
		c.seedDesc = append(c.seedDesc, descriptors...)
	}
}
