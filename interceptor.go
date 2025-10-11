package apideprecation

import (
	"context"
	"maps"
	"slices"
	"strconv"

	_ "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type Metrics struct {
	cfg     *config
	builder *planBuilder

	labelsExtractor   labelsExtractor
	exemplarExtractor labelsExtractor

	onDeprecatedField onDeprecatedFieldFunc
	onDeprecatedEnum  onDeprecatedEnumFunc

	deprecatedFieldUsed *prometheus.CounterVec
	deprecatedEnumUsed  *prometheus.CounterVec
}

func NewMetrics(opts ...Option) *Metrics {
	cfg := &config{}
	for _, opt := range opts {
		opt(cfg)
	}

	builder := newPlanBuilder(cfg.seedDesc)

	labelsExtractor := cfg.extraLabels.extractor()
	exemplarExtractor := cfg.exemplars.extractor()

	defaultLabels := []string{"grpc_type", "grpc_service", "grpc_method", "field"}

	fieldLabels := append(append(defaultLabels, "field_presence"), labelsExtractor.field.labels...)
	enumLabels := append(append(defaultLabels, "enum_value", "enum_number"), labelsExtractor.enum.labels...)

	return &Metrics{
		cfg:               cfg,
		builder:           builder,
		labelsExtractor:   labelsExtractor,
		exemplarExtractor: exemplarExtractor,
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
			m.observe(ctx, msg, interceptors.NewServerCallMeta(info.FullMethod, nil, req))
		}
		return handler(ctx, req)
	}
}

func (m *Metrics) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		meta := interceptors.NewServerCallMeta(info.FullMethod, info, nil)
		return handler(srv, &wrappedServerStream{ServerStream: ss, metrics: m, meta: meta})
	}
}

type wrappedServerStream struct {
	grpc.ServerStream
	metrics *Metrics
	meta    interceptors.CallMeta
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

func (m *Metrics) observe(ctx context.Context, req proto.Message, meta interceptors.CallMeta) {
	typ, service, method := string(meta.Typ), meta.Service, meta.Method

	msg := req.ProtoReflect()
	plan := m.builder.LoadOrBuild(msg.Descriptor())
	plan.EvalMessage(msg, meta.Service, meta.Method,
		func(fd protoreflect.FieldDescriptor, fieldFullName, fieldPresence string) {
			labelExtractors := m.labelsExtractor.field.funcs

			lvs := make([]string, 0, 5+len(labelExtractors))
			lvs = append(lvs, typ, service, method, fieldFullName, fieldPresence)
			for _, valueFunc := range labelExtractors {
				lvs = append(lvs, valueFunc(ctx, req, fd))
			}

			exemplarExtractors := m.exemplarExtractor.field

			exemplar := make(prometheus.Labels, len(exemplarExtractors.labels))
			for i, label := range exemplarExtractors.labels {
				exemplar[label] = exemplarExtractors.funcs[i](ctx, req, fd)
			}

			m.incrementWithExemplar(m.deprecatedFieldUsed, lvs, exemplar)
		},
		func(fd protoreflect.FieldDescriptor, fieldFullName, enumValue string, enumNumber int) {
			labelExtractors := m.labelsExtractor.enum.funcs

			lvs := make([]string, 0, 6+len(labelExtractors))
			lvs = append(lvs, typ, service, method, fieldFullName, enumValue, strconv.Itoa(enumNumber))
			for _, valueFunc := range labelExtractors {
				lvs = append(lvs, valueFunc(ctx, req, fd))
			}

			exemplarExtractors := m.exemplarExtractor.enum

			exemplar := make(prometheus.Labels, len(exemplarExtractors.labels))
			for i, label := range exemplarExtractors.labels {
				exemplar[label] = exemplarExtractors.funcs[i](ctx, req, fd)
			}

			m.incrementWithExemplar(m.deprecatedEnumUsed, lvs, exemplar)
		})
}

func (m *Metrics) onDeprecatedFieldFunc( /*TODO*/ ) onDeprecatedFieldFunc {
	return func(fd protoreflect.FieldDescriptor, fieldFullName, fieldPresence string) {
		// TODO
	}
}

func (m *Metrics) onDeprecatedEnumFunc( /*TODO*/ ) onDeprecatedEnumFunc {
	return func(fd protoreflect.FieldDescriptor, fieldFullName, enumValue string, enumNumber int) {
		// TODO
	}
}

func (m *Metrics) incrementWithExemplar(c *prometheus.CounterVec, lvs []string, exemplar prometheus.Labels) {
	c.WithLabelValues(lvs...).(prometheus.ExemplarAdder).AddWithExemplar(1, exemplar)
}

func (el ExtraLabels) extractor() labelsExtractor {
	return labelsExtractor{
		field: el.labelsFuncs(el.Field),
		enum:  el.labelsFuncs(el.Enum),
	}
}

func (el ExtraLabels) labelsFuncs(m map[string]LabelValueFunc) labelsFuncs {
	lfs := labelsFuncs{
		labels: slices.Collect(maps.Keys(m)),
		funcs:  make([]LabelValueFunc, len(m)),
	}
	slices.Sort(lfs.labels)
	for _, label := range lfs.labels {
		lfs.funcs = append(lfs.funcs, m[label])
	}
	return lfs
}

type labelsExtractor struct {
	field labelsFuncs
	enum  labelsFuncs
}

type labelsFuncs struct {
	labels []string
	funcs  []LabelValueFunc
}
