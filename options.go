package apideprecation

import (
	"context"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type options struct {
	extraLabels ExtraLabels
	exemplars   Exemplars
	seedDesc    []protoreflect.MessageDescriptor
}

type ExtraLabels struct {
	Field map[string]LabelValueFunc
	Enum  map[string]LabelValueFunc
}

type Exemplars = ExtraLabels

type LabelValueFunc func(ctx context.Context, msg proto.Message, fd protoreflect.FieldDescriptor) string

type Option func(*options)

func WithExtraLabels(extraLabels ExtraLabels) Option {
	return func(o *options) {
		o.extraLabels = extraLabels
	}
}

func WithExemplars(exemplars Exemplars) Option {
	return func(o *options) {
		o.exemplars = exemplars
	}
}

func WithMessageDescriptors(descriptors ...protoreflect.MessageDescriptor) Option {
	return func(o *options) {
		o.seedDesc = descriptors
	}
}
