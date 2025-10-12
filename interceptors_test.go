//nolint:staticcheck
package apideprecation

import (
	"context"
	"slices"
	"strconv"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	pb "github.com/belo4ya/grpc-api-deprecation/internal/testdata/proto/proto"
)

func TestUnaryServerInterceptor(t *testing.T) {
	ctx := context.Background()
	metrics := NewMetrics()

	type fieldMetric struct {
		val             float64
		field, presence string
	}
	type enumMetric struct {
		val                  float64
		field, value, number string
	}

	type args struct {
		ctx   context.Context
		req   any
		calls int
	}
	type want struct {
		fields []fieldMetric
		enums  []enumMetric
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "deprecated not populated",
			args: args{
				ctx: ctx,
				req: &pb.AllInclusive{
					Scalar:  1,
					Message: &pb.Simple{Field: 1},
					Maps:    &pb.Maps{Scalars: map[string]string{"a": "b"}},
					Lists:   &pb.Lists{Scalars: []int32{1, 2, 3}},
				},
				calls: 1,
			},
			want: want{},
		},
		{
			name: "deprecated fields 1",
			args: args{
				ctx: ctx,
				req: &pb.AllInclusive{
					ScalarOptionalDeprecated: lo.ToPtr[int32](1),
					Message:                  &pb.Simple{Field: 1},
				},
				calls: 1,
			},
			want: want{fields: []fieldMetric{{
				val:   1,
				field: "scalar_optional_deprecated", presence: "explicit",
			}}},
		},
		{
			name: "deprecated fields 2",
			args: args{
				ctx: ctx,
				req: &pb.AllInclusive{
					ScalarDeprecated: 1,
					Message:          &pb.Simple{FieldDeprecated: 1},
					Maps:             &pb.Maps{ScalarsDeprecate: map[string]string{"a": "b"}},
					Lists:            &pb.Lists{Messages: []*pb.Simple{{FieldDeprecated: 1}}},
				},
				calls: 1,
			},
			want: want{fields: []fieldMetric{{
				val:   1,
				field: "scalar_deprecated", presence: "implicit",
			}, {
				val:   1,
				field: "message.field_deprecated", presence: "implicit",
			}, {
				val:   1,
				field: "maps.scalars_deprecate", presence: "implicit",
			}, {
				val:   1,
				field: "lists.messages[].field_deprecated", presence: "implicit",
			}}},
		},
		{
			name: "deprecated enums",
			args: args{
				ctx: ctx,
				req: &pb.AllInclusive{
					Enum: pb.Enum_ENUM_DEPRECATED,
					Lists: &pb.Lists{
						Enums: []pb.Enum{pb.Enum_ENUM_VALUE, pb.Enum_ENUM_DEPRECATED},
					},
					Maps: &pb.Maps{
						Enums: map[string]pb.Enum{"a": pb.Enum_ENUM_VALUE, "b": pb.Enum_ENUM_DEPRECATED},
					},
				},
				calls: 1,
			},
			want: want{enums: []enumMetric{{
				val:   1,
				field: "enum", value: "ENUM_DEPRECATED", number: "2",
			}, {
				val:   1,
				field: "lists.enums", value: "ENUM_DEPRECATED", number: "2",
			}, {
				val:   1,
				field: "maps.enums", value: "ENUM_DEPRECATED", number: "2",
			}}},
		},
		{
			name: "deprecated fields & enums",
			args: args{
				ctx: ctx,
				req: &pb.AllInclusive{
					Scalar:           1,
					ScalarDeprecated: 1,
					Enum:             pb.Enum_ENUM_DEPRECATED,
					Maps:             &pb.Maps{Scalars: map[string]string{"a": "b"}},
				},
				calls: 1,
			},
			want: want{fields: []fieldMetric{{
				val:   1,
				field: "scalar_deprecated", presence: "implicit",
			}}, enums: []enumMetric{{
				val:   1,
				field: "enum", value: "ENUM_DEPRECATED", number: "2",
			}}},
		},
		{
			name: "no initiator",
			args: args{
				ctx:   context.Background(),
				req:   &pb.AllInclusive{ScalarDeprecated: 1},
				calls: 1,
			},
			want: want{fields: []fieldMetric{{
				val:   1,
				field: "scalar_deprecated", presence: "implicit",
			}}},
		},
		{
			name: "multiple calls with deprecated fields, enums & cache hit",
			args: args{
				ctx: ctx,
				req: &pb.AllInclusive{
					ScalarDeprecated:         1,
					ScalarOptionalDeprecated: lo.ToPtr[int32](1),
					Enum:                     pb.Enum_ENUM_DEPRECATED,
				},
				calls: 3,
			},
			want: want{fields: []fieldMetric{{
				val:   3,
				field: "scalar_deprecated", presence: "implicit",
			}, {
				val:   3,
				field: "scalar_optional_deprecated", presence: "explicit",
			}}, enums: []enumMetric{{
				val:   3,
				field: "enum", value: "ENUM_DEPRECATED", number: "2",
			}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics.deprecatedFieldUsed.Reset()
			metrics.deprecatedEnumUsed.Reset()

			interceptor := metrics.UnaryServerInterceptor()
			for range tt.args.calls {
				_, err := interceptor(
					tt.args.ctx, tt.args.req,
					&grpc.UnaryServerInfo{FullMethod: "/t.Service/Method"},
					func(ctx context.Context, req any) (any, error) { return nil, nil },
				)
				assert.NoError(t, err)
			}

			assert.Equal(t, len(tt.want.fields), testutil.CollectAndCount(metrics.deprecatedFieldUsed))
			for _, want := range tt.want.fields {
				c := metrics.deprecatedFieldUsed.WithLabelValues("unary", "t.Service", "Method", want.field, want.presence)
				assert.Equal(t, want.val, testutil.ToFloat64(c))
			}

			assert.Equal(t, len(tt.want.enums), testutil.CollectAndCount(metrics.deprecatedEnumUsed))
			for _, want := range tt.want.enums {
				c := metrics.deprecatedEnumUsed.WithLabelValues("unary", "t.Service", "Method", want.field, want.value, want.number)
				assert.Equal(t, want.val, testutil.ToFloat64(c))
			}
		})
	}
}

func TestUnaryServerInterceptor__correctness(t *testing.T) {
	metrics := NewMetrics()

	service := "t.Service"
	method := "Method"
	interceptor := func(req any) error {
		interceptor := metrics.UnaryServerInterceptor()
		_, err := interceptor(
			context.Background(), req,
			&grpc.UnaryServerInfo{FullMethod: "/" + service + "/" + method},
			func(ctx context.Context, req any) (any, error) { return nil, nil },
		)
		return err
	}

	type fieldMetric struct {
		field, presence string
	}
	type enumMetric struct {
		field, value, number string
	}

	type want struct {
		fields []fieldMetric
		enums  []enumMetric
	}
	tests := []struct {
		name string
		msg  proto.Message
		want want
	}{
		{
			name: "all-inclusive",
			msg: &pb.AllInclusive{
				Scalar:         1,
				ScalarOptional: lo.ToPtr[int32](1),
				Timestamp:      timestamppb.Now(),
				StringValue:    wrapperspb.String("a"),
				Enum:           pb.Enum_ENUM_DEPRECATED,
				OneOf1: &pb.OneOf{
					O: &pb.OneOf_Scalar{Scalar: 1},
				},
				OneOf2: &pb.OneOf{
					O: &pb.OneOf_ScalarDeprecated{ScalarDeprecated: 2},
				},
				Lists: &pb.Lists{
					Scalars: []int32{1, 2, 3},
					Messages: []*pb.Simple{
						{Field: 1, FieldDeprecated: 0},
						{Field: 1, FieldDeprecated: 2},
						{Field: 1, FieldDeprecated: 0},
					},
					Enums:             []pb.Enum{pb.Enum_ENUM_VALUE, pb.Enum_ENUM_DEPRECATED},
					ScalarsDeprecated: []int32{9, 8, 7},
					MessagesDeprecated: []*pb.Simple{
						{Field: 2, FieldDeprecated: 0},
						{Field: 2, FieldDeprecated: 2},
						{Field: 2, FieldDeprecated: 0},
					},
					EnumsDeprecated: []pb.Enum{pb.Enum_ENUM_VALUE, pb.Enum_ENUM_DEPRECATED},
				},
				Maps: &pb.Maps{
					Scalars: map[string]string{"a": "b"},
					Messages: map[string]*pb.Simple{
						"a": {Field: 1, FieldDeprecated: 0},
						"b": {Field: 1, FieldDeprecated: 2},
						"c": {Field: 1, FieldDeprecated: 0},
					},
					ScalarsDeprecate: map[string]string{"a": "b"},
					MessagesDeprecate: map[string]*pb.Simple{
						"a": {Field: 1, FieldDeprecated: 0},
						"b": {Field: 1, FieldDeprecated: 2},
						"c": {Field: 1, FieldDeprecated: 0},
					},
				},
				Message: &pb.Simple{
					Field:           1,
					FieldDeprecated: 2,
				},
				MessageRecursive: &pb.AllInclusive{
					Message: &pb.Simple{
						Field:           1,
						FieldDeprecated: 2,
					},
					MessageDeprecated: &pb.Simple{
						Field:           1,
						FieldDeprecated: 2,
					},
				},
				MessageNestedRecursive: &pb.AllInclusive_NestedRecursive{
					Message: &pb.AllInclusive{
						MessageRecursive: &pb.AllInclusive{
							ScalarDeprecated: 2,
						},
					},
					MessageDeprecated: &pb.AllInclusive{
						MessageRecursive: &pb.AllInclusive{
							ScalarDeprecated: 2,
						},
					},
				},
				ScalarDeprecated:         2,
				ScalarOptionalDeprecated: lo.ToPtr[int32](2),
				TimestampDeprecated:      timestamppb.Now(),
				StringValueDeprecated:    wrapperspb.String("a"),
				EnumDeprecated:           pb.Enum_ENUM_DEPRECATED,
				OneOfDeprecated: &pb.OneOf{
					O: &pb.OneOf_Message{Message: &pb.Simple{
						Field:           1,
						FieldDeprecated: 2,
					}},
				},
				OneOf2Deprecated: &pb.OneOf{
					O: &pb.OneOf_MessageDeprecated{MessageDeprecated: &pb.Simple{
						Field:           1,
						FieldDeprecated: 2,
					}},
				},
				ListsDeprecated: &pb.Lists{
					Scalars: []int32{1, 2, 3},
					Messages: []*pb.Simple{
						{Field: 1, FieldDeprecated: 0},
						{Field: 1, FieldDeprecated: 2},
						{Field: 1, FieldDeprecated: 0},
					},
					Enums:             []pb.Enum{pb.Enum_ENUM_VALUE, pb.Enum_ENUM_DEPRECATED},
					ScalarsDeprecated: []int32{9, 8, 7},
					MessagesDeprecated: []*pb.Simple{
						{Field: 2, FieldDeprecated: 0},
						{Field: 2, FieldDeprecated: 2},
						{Field: 2, FieldDeprecated: 0},
					},
					EnumsDeprecated: []pb.Enum{pb.Enum_ENUM_VALUE, pb.Enum_ENUM_DEPRECATED},
				},
				MapsDeprecated: &pb.Maps{
					Scalars: map[string]string{"a": "b"},
					Messages: map[string]*pb.Simple{
						"a": {Field: 1, FieldDeprecated: 0},
						"b": {Field: 1, FieldDeprecated: 2},
						"c": {Field: 1, FieldDeprecated: 0},
					},
					ScalarsDeprecate: map[string]string{"a": "b"},
					MessagesDeprecate: map[string]*pb.Simple{
						"a": {Field: 1, FieldDeprecated: 0},
						"b": {Field: 1, FieldDeprecated: 2},
						"c": {Field: 1, FieldDeprecated: 0},
					},
				},
				MessageDeprecated: &pb.Simple{
					Field:           1,
					FieldDeprecated: 2,
				},
				MessageRecursiveDeprecated: &pb.AllInclusive{
					Message: &pb.Simple{
						Field:           1,
						FieldDeprecated: 2,
					},
					MessageDeprecated: &pb.Simple{
						Field:           1,
						FieldDeprecated: 2,
					},
				},
				MessageNestedRecursiveDeprecated: &pb.AllInclusive_NestedRecursive{
					Message: &pb.AllInclusive{
						MessageRecursive: &pb.AllInclusive{
							ScalarDeprecated: 2,
						},
					},
					MessageDeprecated: &pb.AllInclusive{
						MessageRecursive: &pb.AllInclusive{
							ScalarDeprecated: 2,
						},
					},
				},
			},
			want: want{
				fields: []fieldMetric{
					{field: "one_of2.scalar_deprecated", presence: "explicit"},
					{field: "lists.messages[].field_deprecated", presence: "implicit"},
					{field: "lists.scalars_deprecated", presence: "implicit"},
					{field: "lists.messages_deprecated", presence: "implicit"},
					{field: "lists.enums_deprecated", presence: "implicit"},
					{field: "maps.messages{}.field_deprecated", presence: "implicit"},
					{field: "maps.scalars_deprecate", presence: "implicit"},
					{field: "maps.messages_deprecate", presence: "implicit"},
					{field: "message.field_deprecated", presence: "implicit"},
					{field: "message_recursive.message.field_deprecated", presence: "implicit"},
					{field: "message_recursive.message_deprecated", presence: "explicit"},
					{field: "message_nested_recursive.message_deprecated", presence: "explicit"},
					{field: "message_nested_recursive.message.message_recursive.scalar_deprecated", presence: "implicit"},
					{field: "scalar_deprecated", presence: "implicit"},
					{field: "scalar_optional_deprecated", presence: "explicit"},
					{field: "timestamp_deprecated", presence: "explicit"},
					{field: "string_value_deprecated", presence: "explicit"},
					{field: "enum_deprecated", presence: "implicit"},
					{field: "one_of_deprecated", presence: "explicit"},
					{field: "one_of2_deprecated", presence: "explicit"},
					{field: "lists_deprecated", presence: "explicit"},
					{field: "maps_deprecated", presence: "explicit"},
					{field: "message_deprecated", presence: "explicit"},
					{field: "message_recursive_deprecated", presence: "explicit"},
					{field: "message_nested_recursive_deprecated", presence: "explicit"},
				},
				enums: []enumMetric{
					{field: "enum", value: "ENUM_DEPRECATED", number: "2"},
					{field: "lists.enums", value: "ENUM_DEPRECATED", number: "2"},
				},
			},
		},
		{
			name: "defaults",
			msg:  &pb.AllInclusive{},
			want: want{},
		},
		{
			name: "deprecated not populated",
			msg: &pb.AllInclusive{
				Scalar:         1,
				ScalarOptional: lo.ToPtr[int32](1),
				Timestamp:      timestamppb.Now(),
				StringValue:    wrapperspb.String("a"),
				Enum:           pb.Enum_ENUM_VALUE,
				OneOf1: &pb.OneOf{
					O: &pb.OneOf_Scalar{Scalar: 1},
				},
				OneOf2: &pb.OneOf{
					O: &pb.OneOf_Message{Message: &pb.Simple{Field: 1}},
				},
				Lists: &pb.Lists{
					Scalars: []int32{1, 2, 3},
					Messages: []*pb.Simple{
						{Field: 1},
						{Field: 1},
						{Field: 1},
					},
					Enums: []pb.Enum{pb.Enum_ENUM_VALUE, pb.Enum_ENUM_VALUE},
				},
				Maps: &pb.Maps{
					Scalars: map[string]string{"a": "b"},
					Messages: map[string]*pb.Simple{
						"a": {Field: 1},
						"b": {Field: 1},
						"c": {Field: 1},
					},
				},
				Message: &pb.Simple{Field: 1},
				MessageRecursive: &pb.AllInclusive{
					Message: &pb.Simple{Field: 1},
				},
				MessageNestedRecursive: &pb.AllInclusive_NestedRecursive{
					Message: &pb.AllInclusive{
						MessageRecursive: &pb.AllInclusive{
							Scalar: 1,
						},
					},
				},
			},
			want: want{},
		},
		{
			name: "only deprecated",
			msg: &pb.AllInclusive{
				ScalarDeprecated:         2,
				ScalarOptionalDeprecated: lo.ToPtr[int32](2),
				TimestampDeprecated:      timestamppb.Now(),
				StringValueDeprecated:    wrapperspb.String("a"),
				EnumDeprecated:           pb.Enum_ENUM_VALUE,
				OneOfDeprecated: &pb.OneOf{
					O: &pb.OneOf_Message{Message: &pb.Simple{
						FieldDeprecated: 2,
					}},
				},
				OneOf2Deprecated: &pb.OneOf{
					O: &pb.OneOf_MessageDeprecated{
						MessageDeprecated: &pb.Simple{FieldDeprecated: 2},
					},
				},
				ListsDeprecated: &pb.Lists{
					ScalarsDeprecated: []int32{9, 8, 7},
					MessagesDeprecated: []*pb.Simple{
						{FieldDeprecated: 1},
						{FieldDeprecated: 2},
					},
				},
				MapsDeprecated: &pb.Maps{
					ScalarsDeprecate: map[string]string{"a": "b"},
					MessagesDeprecate: map[string]*pb.Simple{
						"a": {FieldDeprecated: 1},
						"b": {FieldDeprecated: 2},
					},
				},
				MessageDeprecated: &pb.Simple{FieldDeprecated: 2},
				MessageRecursiveDeprecated: &pb.AllInclusive{
					Message:           &pb.Simple{FieldDeprecated: 2},
					MessageDeprecated: &pb.Simple{FieldDeprecated: 2},
				},
				MessageNestedRecursiveDeprecated: &pb.AllInclusive_NestedRecursive{
					MessageDeprecated: &pb.AllInclusive{
						MessageRecursive: &pb.AllInclusive{ScalarDeprecated: 2},
					},
				},
			},
			want: want{
				fields: []fieldMetric{
					{field: "scalar_deprecated", presence: "implicit"},
					{field: "scalar_optional_deprecated", presence: "explicit"},
					{field: "timestamp_deprecated", presence: "explicit"},
					{field: "string_value_deprecated", presence: "explicit"},
					{field: "enum_deprecated", presence: "implicit"},
					{field: "one_of_deprecated", presence: "explicit"},
					{field: "one_of2_deprecated", presence: "explicit"},
					{field: "lists_deprecated", presence: "explicit"},
					{field: "maps_deprecated", presence: "explicit"},
					{field: "message_deprecated", presence: "explicit"},
					{field: "message_recursive_deprecated", presence: "explicit"},
					{field: "message_nested_recursive_deprecated", presence: "explicit"},
				},
				enums: []enumMetric{},
			},
		},
		{
			name: "lists: no deprecated",
			msg: &pb.Lists{
				Scalars:  []int32{1, 2, 3},
				Messages: []*pb.Simple{{Field: 1}, {Field: 2}},
				Enums:    []pb.Enum{pb.Enum_ENUM_VALUE, pb.Enum_ENUM_VALUE},
			},
			want: want{},
		},
		{
			name: "lists: with deprecated values",
			msg: &pb.Lists{
				Messages: []*pb.Simple{{}, {FieldDeprecated: 1}, {Field: 2}},
				Enums:    []pb.Enum{pb.Enum_ENUM_VALUE, pb.Enum_ENUM_DEPRECATED},
			},
			want: want{
				fields: []fieldMetric{
					{field: "messages[].field_deprecated", presence: "implicit"},
				},
				enums: []enumMetric{
					{field: "enums", value: "ENUM_DEPRECATED", number: "2"},
				},
			},
		},
		{
			name: "maps: no deprecated",
			msg: &pb.Maps{
				Scalars: map[string]string{"a": "b", "c": "d"},
				Messages: map[string]*pb.Simple{
					"a": {Field: 1},
					"b": {Field: 2},
				},
				Enums: map[string]pb.Enum{
					"a": pb.Enum_ENUM_VALUE,
					"b": pb.Enum_ENUM_VALUE,
				},
			},
		},
		{
			name: "maps: with deprecated values",
			msg: &pb.Maps{
				Scalars: map[string]string{"a": "b", "c": "d"},
				Messages: map[string]*pb.Simple{
					"a": {},
					"b": {FieldDeprecated: 1},
					"c": {Field: 2},
				},
				Enums: map[string]pb.Enum{
					"a": pb.Enum_ENUM_VALUE,
					"b": pb.Enum_ENUM_DEPRECATED,
				},
			},
			want: want{
				fields: []fieldMetric{
					{field: "messages{}.field_deprecated", presence: "implicit"},
				},
				enums: []enumMetric{
					{field: "enums", value: "ENUM_DEPRECATED", number: "2"},
				},
			},
		},
		{
			name: "nested message with deprecated",
			msg: &pb.AllInclusive{
				MessageRecursive: &pb.AllInclusive{
					MessageRecursive: &pb.AllInclusive{
						ScalarDeprecated: 1,
					},
					MessageNestedRecursive: &pb.AllInclusive_NestedRecursive{
						Message: &pb.AllInclusive{
							MessageDeprecated: &pb.Simple{},
						},
					},
				},
			},
			want: want{
				fields: []fieldMetric{
					{field: "message_recursive.message_recursive.scalar_deprecated", presence: "implicit"},
					{field: "message_recursive.message_nested_recursive.message.message_deprecated", presence: "explicit"},
				},
				enums: []enumMetric{},
			},
		},
		{
			name: "nested message without deprecated",
			msg: &pb.AllInclusive{
				MessageRecursive: &pb.AllInclusive{
					MessageRecursive: &pb.AllInclusive{
						Scalar: 1,
					},
					MessageNestedRecursive: &pb.AllInclusive_NestedRecursive{
						Message: &pb.AllInclusive{
							Message: &pb.Simple{},
						},
					},
				},
			},
			want: want{},
		},
		{
			name: "only top-level deprecated",
			msg: &pb.AllInclusive{
				MessageRecursiveDeprecated: &pb.AllInclusive{
					ScalarDeprecated:  1,
					MessageDeprecated: &pb.Simple{},
				},
			},
			want: want{
				fields: []fieldMetric{
					{field: "message_recursive_deprecated", presence: "explicit"},
				},
				enums: []enumMetric{},
			},
		},
		{
			name: "without deprecated annotations",
			msg: &pb.WithoutDeprecated{
				Scalar:  1,
				List:    []int32{1, 2, 3},
				Map:     map[string]string{"a": "b"},
				Message: &pb.WithoutDeprecated_Simple{Field: 1},
			},
			want: want{},
		},
		{
			name: "field presence: defaults",
			msg:  &pb.TypesPresence{},
			want: want{},
		},
		{
			name: "field presence: populated",
			msg: &pb.TypesPresence{
				Bool:                true,
				Float:               1,
				Double:              1,
				Bytes:               []byte{'b'},
				String_:             "a",
				Int32:               1,
				Int64:               1,
				Sint32:              1,
				Sint64:              1,
				Sfixed32:            1,
				Sfixed64:            1,
				Uint32:              1,
				Uint64:              1,
				Fixed32:             1,
				Fixed64:             1,
				Enum:                pb.Enum_ENUM_VALUE,
				OneOf:               &pb.OneOf{},
				Repeated:            []int32{1, 2, 3},
				Map:                 map[string]string{"a": "b"},
				Message:             &pb.Simple{},
				BoolOptional:        lo.ToPtr(false),
				FloatOptional:       lo.ToPtr[float32](1),
				DoubleOptional:      lo.ToPtr[float64](1),
				BytesOptional:       []byte{'b'},
				StringOptional:      lo.ToPtr("a"),
				Int32Optional:       lo.ToPtr[int32](1),
				Int64Optional:       lo.ToPtr[int64](1),
				Sint32Optional:      lo.ToPtr[int32](1),
				Sint64Optional:      lo.ToPtr[int64](1),
				Sfixed32Optional:    lo.ToPtr[int32](1),
				Sfixed64Optional:    lo.ToPtr[int64](1),
				Uint32Optional:      lo.ToPtr[uint32](1),
				Uint64Optional:      lo.ToPtr[uint64](1),
				Fixed32Optional:     lo.ToPtr[uint32](1),
				Fixed64Optional:     lo.ToPtr[uint64](1),
				EnumOptional:        lo.ToPtr(pb.Enum_ENUM_VALUE),
				OneOfOptional:       &pb.OneOf{},
				MessageOptional:     &pb.Simple{},
				StringValue:         wrapperspb.String("a"),
				Timestamp:           timestamppb.Now(),
				StringValueOptional: wrapperspb.String("a"),
				TimestampOptional:   timestamppb.Now(),
			},
			want: want{
				fields: []fieldMetric{
					{field: "bool", presence: "implicit"},
					{field: "float", presence: "implicit"},
					{field: "double", presence: "implicit"},
					{field: "bytes", presence: "implicit"},
					{field: "string", presence: "implicit"},
					{field: "int32", presence: "implicit"},
					{field: "int64", presence: "implicit"},
					{field: "sint32", presence: "implicit"},
					{field: "sint64", presence: "implicit"},
					{field: "sfixed32", presence: "implicit"},
					{field: "sfixed64", presence: "implicit"},
					{field: "uint32", presence: "implicit"},
					{field: "uint64", presence: "implicit"},
					{field: "fixed32", presence: "implicit"},
					{field: "fixed64", presence: "implicit"},
					{field: "enum", presence: "implicit"},
					{field: "one_of", presence: "explicit"},
					{field: "repeated", presence: "implicit"},
					{field: "map", presence: "implicit"},
					{field: "message", presence: "explicit"},
					{field: "bool_optional", presence: "explicit"},
					{field: "float_optional", presence: "explicit"},
					{field: "double_optional", presence: "explicit"},
					{field: "bytes_optional", presence: "explicit"},
					{field: "string_optional", presence: "explicit"},
					{field: "int32_optional", presence: "explicit"},
					{field: "int64_optional", presence: "explicit"},
					{field: "sint32_optional", presence: "explicit"},
					{field: "sint64_optional", presence: "explicit"},
					{field: "sfixed32_optional", presence: "explicit"},
					{field: "sfixed64_optional", presence: "explicit"},
					{field: "uint32_optional", presence: "explicit"},
					{field: "uint64_optional", presence: "explicit"},
					{field: "fixed32_optional", presence: "explicit"},
					{field: "fixed64_optional", presence: "explicit"},
					{field: "enum_optional", presence: "explicit"},
					{field: "one_of_optional", presence: "explicit"},
					{field: "message_optional", presence: "explicit"},
					{field: "string_value", presence: "explicit"},
					{field: "timestamp", presence: "explicit"},
					{field: "string_value_optional", presence: "explicit"},
					{field: "timestamp_optional", presence: "explicit"},
				},
				enums: []enumMetric{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics.deprecatedFieldUsed.Reset()
			metrics.deprecatedEnumUsed.Reset()
			assert.NoError(t, interceptor(tt.msg))

			assert.Equal(t, len(tt.want.fields), testutil.CollectAndCount(metrics.deprecatedFieldUsed))
			for _, want := range tt.want.fields {
				c := metrics.deprecatedFieldUsed.WithLabelValues("unary", service, method, want.field, want.presence)
				assert.Equal(t, float64(1), testutil.ToFloat64(c))
			}

			assert.Equal(t, len(tt.want.enums), testutil.CollectAndCount(metrics.deprecatedEnumUsed))
			for _, want := range tt.want.enums {
				c := metrics.deprecatedEnumUsed.WithLabelValues("unary", service, method, want.field, want.value, want.number)
				assert.Equal(t, float64(1), testutil.ToFloat64(c))
			}
		})
	}
}

func TestTestUnaryServerInterceptor__maxItemsPerCollection(t *testing.T) {
	metrics := NewMetrics()

	service := "t.Service"
	method := "Method"
	interceptor := func(req any) error {
		metrics.deprecatedFieldUsed.Reset()
		hitMaxItemsPerCollection.Reset()

		interceptor := metrics.UnaryServerInterceptor()
		_, err := interceptor(
			context.Background(), req,
			&grpc.UnaryServerInfo{FullMethod: "/" + service + "/" + method},
			func(ctx context.Context, req any) (any, error) { return nil, nil },
		)
		return err
	}

	tests := []struct {
		name       string
		msg        proto.Message
		callAssert func(func(prometheus.Collector))
	}{
		{
			name: "hit maxItemsPerCollection: list",
			msg: &pb.Lists{Messages: append(
				slices.Repeat([]*pb.Simple{{}}, 10*maxItemsPerCollection),
				&pb.Simple{FieldDeprecated: 1},
			)},
			callAssert: func(assertHit func(prometheus.Collector)) {
				c := hitMaxItemsPerCollection.WithLabelValues("unary", service, method, "messages", "repeated", strconv.Itoa(maxItemsPerCollection))
				assertHit(c)
			},
		},
		{
			name: "hit maxItemsPerCollection: map",
			msg: func() proto.Message {
				msg := &pb.Maps{Messages: map[string]*pb.Simple{}}
				for i := range 10 * maxItemsPerCollection {
					msg.Messages[strconv.Itoa(i)] = &pb.Simple{}
				}
				return msg
			}(),
			callAssert: func(assertHit func(prometheus.Collector)) {
				c := hitMaxItemsPerCollection.WithLabelValues("unary", service, method, "messages", "map", strconv.Itoa(maxItemsPerCollection))
				assertHit(c)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NoError(t, interceptor(tt.msg))
			tt.callAssert(func(c prometheus.Collector) {
				assert.Equal(t, float64(1), testutil.ToFloat64(c))
			})
			assert.Equal(t, 0, testutil.CollectAndCount(metrics.deprecatedFieldUsed))
			assert.Equal(t, 0, testutil.CollectAndCount(metrics.deprecatedEnumUsed))
		})
	}
}
