package apideprecation

import (
	"context"
	"strconv"

	_ "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type Metrics struct {
	cfg     *config
	builder *planBuilder

	extraLabels compiledLabels
	exemplar    compiledLabels

	deprecatedFieldUsed *prometheus.CounterVec
	deprecatedEnumUsed  *prometheus.CounterVec
}

func NewMetrics(opts ...Option) *Metrics {
	cfg := &config{}
	for _, opt := range opts {
		opt(cfg)
	}

	defaultLabels := []string{"grpc_type", "grpc_service", "grpc_method", "field"}

	extraLabels := cfg.extraLabels.compile()
	fieldLabels := append(append(defaultLabels, "field_presence"), extraLabels.fieldLabels...)
	enumLabels := append(append(defaultLabels, "enum_value", "enum_number"), extraLabels.enumLabels...)

	return &Metrics{
		cfg:         cfg,
		builder:     newPlanBuilder(cfg.seedDesc),
		extraLabels: extraLabels,
		exemplar:    cfg.exemplar.compile(),
		deprecatedFieldUsed: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "grpc_deprecated_field_used_total",
			Help: "Count of requests using deprecated fields (proto field option deprecated=true).",
		}, fieldLabels),
		deprecatedEnumUsed: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "grpc_deprecated_enum_used_total",
			Help: "Count of requests using deprecated enum values (proto enum value option deprecated=true).",
		}, enumLabels),
	}
}

func (m *Metrics) Describe(ch chan<- *prometheus.Desc) {
	m.deprecatedFieldUsed.Describe(ch)
	m.deprecatedEnumUsed.Describe(ch)
}

func (m *Metrics) Collect(ch chan<- prometheus.Metric) {
	m.deprecatedFieldUsed.Collect(ch)
	m.deprecatedEnumUsed.Collect(ch)
}

func (m *Metrics) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if msg, ok := req.(proto.Message); ok {
			m.observe(ctx, msg, newCallMeta(info.FullMethod, nil))
		}
		return handler(ctx, req)
	}
}

func (m *Metrics) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		return handler(srv, &wrappedServerStream{ServerStream: ss, metrics: m, meta: newCallMeta(info.FullMethod, info)})
	}
}

type wrappedServerStream struct {
	grpc.ServerStream
	metrics *Metrics
	meta    callMeta
}

func (s *wrappedServerStream) RecvMsg(m any) error {
	if err := s.ServerStream.RecvMsg(m); err != nil {
		return err
	}
	if msg, ok := m.(proto.Message); ok {
		s.metrics.observe(s.Context(), msg, s.meta)
	}
	return nil
}

func (m *Metrics) observe(ctx context.Context, req proto.Message, meta callMeta) {
	typ, service, method := meta.Type, meta.Service, meta.Method

	// TODO: sync.Pool can slightly speed up the onDeprecatedField and onDeprecatedEnum functions.

	msg := req.ProtoReflect()
	plan := m.builder.LoadOrBuild(msg.Descriptor())
	plan.EvalMessage(msg, meta.Service, meta.Method,
		func(fd protoreflect.FieldDescriptor, fieldFullName, fieldPresence string) {
			base := []string{typ, service, method, fieldFullName, fieldPresence}
			lvs := m.buildLabelValues(base, m.extraLabels.fieldValues, ctx, req, fd)
			exemplar := m.buildExemplar(m.exemplar.fieldLabels, m.exemplar.fieldValues, ctx, req, fd)
			m.increment(m.deprecatedFieldUsed, lvs, exemplar)
		},
		func(fd protoreflect.FieldDescriptor, fieldFullName, enumValue string, enumNumber int) {
			base := []string{typ, service, method, fieldFullName, enumValue, strconv.Itoa(enumNumber)}
			lvs := m.buildLabelValues(base, m.extraLabels.enumValues, ctx, req, fd)
			exemplar := m.buildExemplar(m.exemplar.enumLabels, m.exemplar.enumValues, ctx, req, fd)
			m.increment(m.deprecatedEnumUsed, lvs, exemplar)
		})
}

func (m *Metrics) buildLabelValues(
	base []string,
	valFuncs []LabelValueFunc,
	ctx context.Context, req proto.Message, fd protoreflect.FieldDescriptor,
) []string {
	lvs := make([]string, 0, len(base)+len(valFuncs))
	lvs = append(lvs, base...)
	for _, valF := range valFuncs {
		lvs = append(lvs, valF(ctx, req, fd))
	}
	return lvs
}

func (m *Metrics) buildExemplar(
	labels []string,
	valFuncs []LabelValueFunc,
	ctx context.Context, req proto.Message, fd protoreflect.FieldDescriptor,
) prometheus.Labels {
	if len(labels) == 0 {
		return nil
	}
	exemplar := make(prometheus.Labels, len(labels))
	for i, label := range labels {
		exemplar[label] = valFuncs[i](ctx, req, fd)
	}
	return exemplar
}

func (m *Metrics) increment(c *prometheus.CounterVec, lvs []string, exemplar prometheus.Labels) {
	if exemplar == nil {
		c.WithLabelValues(lvs...).Inc()
	} else {
		c.WithLabelValues(lvs...).(prometheus.ExemplarAdder).AddWithExemplar(1, exemplar)
	}
}

type compiledLabels struct {
	fieldLabels []string
	fieldValues []LabelValueFunc
	enumLabels  []string
	enumValues  []LabelValueFunc
}

func (s LabelSet) compile() compiledLabels {
	compiled := compiledLabels{
		fieldLabels: make([]string, 0, len(s.Field)),
		fieldValues: make([]LabelValueFunc, 0, len(s.Field)),
		enumLabels:  make([]string, 0, len(s.Enum)),
		enumValues:  make([]LabelValueFunc, 0, len(s.Enum)),
	}
	for _, label := range s.Field {
		compiled.fieldLabels = append(compiled.fieldLabels, label.Name)
		compiled.fieldValues = append(compiled.fieldValues, label.Value)
	}
	for _, label := range s.Enum {
		compiled.enumLabels = append(compiled.enumLabels, label.Name)
		compiled.enumValues = append(compiled.enumValues, label.Value)
	}
	return compiled
}
