package apideprecation

import (
	"context"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// Metrics represents a collection of metrics to be registered on a
// Prometheus metrics registry for a gRPC server.
type Metrics struct {
	cfg         *config
	extraLabels compiledLabels
	exemplar    compiledLabels

	methodReporter *methodReporter
	fieldReporter  *fieldReporter

	deprecatedMethodUsed *prometheus.CounterVec
	deprecatedFieldUsed  *prometheus.CounterVec
	deprecatedEnumUsed   *prometheus.CounterVec
}

// NewMetrics returns a new Metrics object that has server interceptor methods.
// NOTE: Remember to register Metrics object by using prometheus registry, e.g. prometheus.MustRegister(metrics).
func NewMetrics(opts ...Option) *Metrics {
	cfg := &config{}
	for _, opt := range opts {
		opt(cfg)
	}

	defaultLabels := []string{"grpc_type", "grpc_service", "grpc_method"}

	extraLabels := cfg.extraLabels.compile()
	methodLabels := append(defaultLabels, extraLabels.fieldLabels...)
	fieldLabels := append(append(defaultLabels, "field", "field_presence"), extraLabels.fieldLabels...)
	enumLabels := append(append(defaultLabels, "field", "enum_value", "enum_number"), extraLabels.enumLabels...)

	return &Metrics{
		cfg:            cfg,
		extraLabels:    extraLabels,
		exemplar:       cfg.exemplar.compile(),
		methodReporter: newMethodReporter(),
		fieldReporter:  newFieldReporter(cfg.seedDesc),
		deprecatedMethodUsed: prometheus.NewCounterVec(
			cfg.counterOpts.apply(prometheus.CounterOpts{
				Name: "grpc_deprecated_method_used_total",
				Help: "Count of calls to deprecated RPC methods (proto field option deprecated=true).",
			}), methodLabels),
		deprecatedFieldUsed: prometheus.NewCounterVec(
			cfg.counterOpts.apply(prometheus.CounterOpts{
				Name: "grpc_deprecated_field_used_total",
				Help: "Count of requests using deprecated fields (proto field option deprecated=true).",
			}), fieldLabels),
		deprecatedEnumUsed: prometheus.NewCounterVec(
			cfg.counterOpts.apply(prometheus.CounterOpts{
				Name: "grpc_deprecated_enum_used_total",
				Help: "Count of requests using deprecated enum values (proto enum value option deprecated=true).",
			}), enumLabels),
	}
}

// Describe implements prometheus.Collector.
func (m *Metrics) Describe(ch chan<- *prometheus.Desc) {
	m.deprecatedFieldUsed.Describe(ch)
	m.deprecatedEnumUsed.Describe(ch)
}

// Collect implements prometheus.Collector.
func (m *Metrics) Collect(ch chan<- prometheus.Metric) {
	m.deprecatedFieldUsed.Collect(ch)
	m.deprecatedEnumUsed.Collect(ch)
}

// UnaryServerInterceptor is a gRPC server-side interceptor
// that provides deprecated API usage tracking for Unary RPCs.
func (m *Metrics) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if msg, ok := req.(proto.Message); ok {
			m.observe(ctx, msg, newCallMeta(info.FullMethod, nil))
		}
		return handler(ctx, req)
	}
}

// StreamServerInterceptor is a gRPC server-side interceptor
// that provides deprecated API usage tracking for Streaming RPCs.
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

	methodDeprecated := m.methodReporter.Report(meta, func() {
		base := []string{typ, service, method}
		lvs := base
		exemplar := map[string]string{}
		m.increment(m.deprecatedMethodUsed, lvs, exemplar)
	})
	if methodDeprecated {
		return
	}

	// TODO: sync.Pool can slightly speed up the onDeprecatedField and onDeprecatedEnum functions.

	m.fieldReporter.Report(req.ProtoReflect(), meta,
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
