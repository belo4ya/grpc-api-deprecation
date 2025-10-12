package apideprecation

import (
	"context"

	grpcprom "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type config struct {
	extraLabels LabelSet
	exemplar    ExemplarSet
	seedDesc    []protoreflect.MessageDescriptor
	counterOpts counterOptions
}

// LabelSet defines ordered dynamic labels that are appended to the default metric labels.
// Order is preserved as provided by the user.
type LabelSet struct {
	Field []Label
	Enum  []Label
}

// ExemplarSet defines ordered exemplar label extractors. Names are used as exemplar keys.
type ExemplarSet = LabelSet

// Label binds a Prometheus label name to a value extractor function.
type Label struct {
	Name  string
	Value LabelValueFunc
}

// LabelValueFunc extracts a label value from the current call context,
// request message, and the deprecated field descriptor.
// Implementations should be fast and allocation-conscious.
type LabelValueFunc func(ctx context.Context, msg proto.Message, fd protoreflect.FieldDescriptor) string

type Option func(*config)

// WithExtraLabels sets additional metric labels for deprecated field and enum
// observations. Labels are appended after the default labels in the order they are provided.
func WithExtraLabels(extraLabels LabelSet) Option {
	return func(c *config) {
		c.extraLabels = extraLabels
	}
}

// WithExemplar sets exemplar label extractors for deprecated field and enum
// observations. Exemplars are added only if supported by the Counter.
func WithExemplar(exemplar ExemplarSet) Option {
	return func(c *config) {
		c.exemplar = exemplar
	}
}

// WithMessageDescriptors allows warming up the Metrics cache with message
// descriptors that are expected to be validated. Messages included transitively
// (i.e., fields with message values) are automatically handled.
func WithMessageDescriptors(descriptors ...protoreflect.MessageDescriptor) Option {
	return func(c *config) {
		c.seedDesc = descriptors
	}
}

//func WithPrewarm(servers ...reflection.ServiceInfoProvider) Option {
//	// TODO: syntax sugar, which will be more useful than WithMessageDescriptors in most cases
//	panic("implement me")
//}

// WithCounterOptions sets counter options.
// Notice: This API is EXPERIMENTAL and may be changed or removed in a later release.
func WithCounterOptions(opts ...CounterOption) Option {
	return func(c *config) {
		c.counterOpts = opts
	}
}

// CounterOption lets you add options to Counter metrics using With* funcs.
// Notice: This API is EXPERIMENTAL and may be changed or removed in a later release.
type CounterOption = grpcprom.CounterOption

type counterOptions []CounterOption

func (o counterOptions) apply(opts prometheus.CounterOpts) prometheus.CounterOpts {
	for _, f := range o {
		f(&opts)
	}
	return opts
}

// WithConstLabels allows you to add ConstLabels to Counter metrics.
// Notice: This API is EXPERIMENTAL and may be changed or removed in a later release.
var WithConstLabels = grpcprom.WithConstLabels

// WithSubsystem allows you to add a Subsystem to Counter metrics.
// Notice: This API is EXPERIMENTAL and may be changed or removed in a later release.
var WithSubsystem = grpcprom.WithSubsystem

// WithNamespace allows you to add a Namespace to Counter metrics.
// Notice: This API is EXPERIMENTAL and may be changed or removed in a later release.
var WithNamespace = grpcprom.WithNamespace
