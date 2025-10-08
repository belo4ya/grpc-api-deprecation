package apideprecation

import (
	"context"
	"maps"
	"slices"
	"strconv"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var (
	deprecatedFieldUsed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grpc_deprecated_field_used_total",
			Help: "Count of requests using deprecated fields (proto field option deprecated=true).",
		},
		[]string{"grpc_type", "grpc_service", "grpc_method", "field", "field_presence"},
	)
	deprecatedEnumUsed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grpc_deprecated_enum_used_total",
			Help: "Count of requests using deprecated enum values (proto enum value option deprecated=true).",
		},
		[]string{"grpc_type", "grpc_service", "grpc_method", "field", "enum_value", "enum_number"},
	)
)

type Metrics struct {
	cfg                 *config
	builder             *planBuilder
	fieldExtractors     labelsExtractor
	enumExtractors      labelsExtractor
	deprecatedFieldUsed *prometheus.CounterVec
	deprecatedEnumUsed  *prometheus.CounterVec
}

func NewMetrics(opts ...Option) *Metrics {
	cfg := &config{}
	for _, opt := range opts {
		opt(cfg)
	}

	builder := newPlanBuilder(cfg.seedDesc)

	defaultLabels := []string{"grpc_type", "grpc_service", "grpc_method", "field"}
	fieldExtraLabels := cfg.extraLabelsExtractor.field.labels
	enumExtraLabels := cfg.extraLabelsExtractor.enum.labels

	fieldLabels := append(append(defaultLabels, "field_presence"), fieldExtraLabels...)
	enumLabels := append(append(defaultLabels, "enum_value", "enum_number"), enumExtraLabels...)

	return &Metrics{
		cfg:     cfg,
		builder: builder,
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
			extraLabels := m.cfg.extraLabelsExtractor.field.labels
			extraLabelFuncs := m.cfg.extraLabelsExtractor.field.funcs

			lvs := make([]string, 0, 5+len(extraLabels))
			lvs = append(lvs, typ, service, method, fieldFullName, fieldPresence)
			for _, valueFunc := range extraLabelFuncs {
				lvs = append(lvs, valueFunc(ctx, req, fd))
			}

			exemplar := make(prometheus.Labels, len(m.cfg.exemplars.Field))
			for label, value := range m.cfg.exemplars.Field {
				exemplar[label] = value(ctx, req, fd)
			}

			m.incrementWithExemplar(m.deprecatedFieldUsed, lvs, exemplar)
		},
		func(fd protoreflect.FieldDescriptor, fieldFullName, enumValue string, enumNumber int) {
			extraLabels := m.cfg.extraLabelsExtractor.enum.labels
			//extraLabelFuncs := m.cfg.extraLabelsExtractor.enum.funcs

			lvs := make([]string, 0, 6+len(extraLabels))
			lvs = append(lvs, typ, service, method, fieldFullName, enumValue, strconv.Itoa(enumNumber))
			//for _, label := range m.enumExtraLabels {
			//	value := m.cfg.extraLabels.Enum[label](ctx, req, fd)
			//	lvs = append(lvs, value)
			//}
			exemplar := make(prometheus.Labels, len(m.cfg.exemplars.Enum))
			//for label, value := range m.cfg.exemplars.Enum {
			//	exemplar[label] = value(ctx, req, fd)
			//}
			m.incrementWithExemplar(m.deprecatedEnumUsed, lvs, exemplar)
		})
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

func (el ExtraLabels) labelsFuncs(m map[string]labelValueFunc) labelsFuncs {
	lfs := labelsFuncs{
		labels: slices.Collect(maps.Keys(m)),
		funcs:  make([]labelValueFunc, len(m)),
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
	funcs  []labelValueFunc
}
