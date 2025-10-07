package apideprecation

import (
	"context"
	"strconv"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
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

func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	r := reporter{builder: newPlanBuilder()}
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		msg, ok := req.(proto.Message)
		if ok {
			r.Report(ctx, msg, interceptors.NewServerCallMeta(info.FullMethod, nil, req))
		}

		return handler(ctx, req)
	}
}

func StreamServerInterceptor() grpc.StreamServerInterceptor {
	r := reporter{builder: newPlanBuilder()}
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		meta := interceptors.NewServerCallMeta(info.FullMethod, info, nil)
		return handler(srv, &wrappedServerStream{ServerStream: ss, reporter: r, meta: meta})
	}
}

type wrappedServerStream struct {
	grpc.ServerStream
	reporter reporter
	meta     interceptors.CallMeta
}

func (s *wrappedServerStream) RecvMsg(m any) error {
	if err := s.ServerStream.RecvMsg(m); err != nil {
		return err
	}
	if msg, ok := m.(proto.Message); ok {
		s.reporter.Report(s.Context(), msg, s.meta)
	}
	return nil
}

type reporter struct {
	builder *planBuilder
}

func (r reporter) Report(_ context.Context, req proto.Message, meta interceptors.CallMeta) {
	typ, service, method := string(meta.Typ), meta.Service, meta.Method

	msg := req.ProtoReflect()
	plan := r.builder.LoadOrBuild(msg.Descriptor())
	plan.EvalMessage(msg, meta.Service, meta.Method,
		func(fieldFullName, fieldPresence string) {
			deprecatedFieldUsed.WithLabelValues(typ, service, method, fieldFullName, fieldPresence).Inc()
		},
		func(fieldFullName, enumValue string, enumNumber int) {
			deprecatedEnumUsed.WithLabelValues(typ, service, method, fieldFullName, enumValue, strconv.Itoa(enumNumber)).Inc()
		})
}
