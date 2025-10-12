package apideprecation_test

import (
	"context"
	"net"

	apideprecation "github.com/belo4ya/grpc-api-deprecation"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func Example() {
	extraLabels := []apideprecation.Label{{
		Name: "project_id",
		Value: func(ctx context.Context, _ proto.Message, _ apideprecation.CallMeta, _ protoreflect.MethodDescriptor, _ protoreflect.FieldDescriptor) string {
			projectID, _ := ctx.Value("project_id").(string)
			return projectID
		},
	}}

	exemplar := []apideprecation.Label{{
		Name: "user_id",
		Value: func(ctx context.Context, _ proto.Message, _ apideprecation.CallMeta, _ protoreflect.MethodDescriptor, _ protoreflect.FieldDescriptor) string {
			userID, _ := ctx.Value("user_id").(string)
			return userID
		},
	}}

	metrics := apideprecation.NewMetrics(
		apideprecation.WithExtraLabels(apideprecation.LabelSet{Method: extraLabels, Field: extraLabels, Enum: extraLabels}),
		apideprecation.WithExemplar(apideprecation.ExemplarSet{Method: exemplar, Field: exemplar, Enum: exemplar}),
	)
	prometheus.MustRegister(metrics)

	srv := grpc.NewServer(grpc.UnaryInterceptor(metrics.UnaryServerInterceptor()))

	_ = srv.Serve(&net.TCPListener{})
}
