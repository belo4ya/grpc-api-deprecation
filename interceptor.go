package apideprecation

import (
	"context"
	"fmt"
	"strconv"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

//nolint:exhaustruct
var (
	deprecatedFieldUsed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grpc_deprecated_field_used_total",
			Help: "Count of requests using deprecated fields (proto field option deprecated=true).",
		},
		[]string{"grpc_service", "grpc_method", "field", "field_presence", "project_id", "user_id"},
	)
	deprecatedEnumUsed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grpc_deprecated_enum_used_total",
			Help: "Count of requests using deprecated enum values (proto enum value option deprecated=true).",
		},
		[]string{"grpc_service", "grpc_method", "field", "enum_value", "enum_number", "project_id", "user_id"},
	)
)

func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	r := reporter{builder: newPlanBuilder()}
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		msg, ok := req.(proto.Message)
		if ok {
			meta := interceptors.NewServerCallMeta(info.FullMethod, nil, req)
			r.Report(ctx, msg, meta.Service, meta.Method)
		}

		return handler(ctx, req)
	}
}

type reporter struct {
	builder *planBuilder
}

func (r reporter) Report(_ context.Context, req proto.Message, service, method string) {
	var projectID, userID string

	msg := req.ProtoReflect()
	plan := r.builder.loadOrBuild(msg.Descriptor())
	plan.EvalMessage(msg, service, method,
		func(fieldFullName, fieldPresence string) {
			deprecatedFieldUsed.WithLabelValues(service, method, fieldFullName, fieldPresence, projectID, userID).Inc()
		},
		func(fieldFullName, enumValue string, enumNumber int) {
			fmt.Println(fieldFullName, enumValue, enumNumber)
			deprecatedEnumUsed.WithLabelValues(service, method, fieldFullName, enumValue, strconv.Itoa(enumNumber), projectID, userID).Inc()
		},
	)
}
