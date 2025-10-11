package apideprecation

import (
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors"
	"google.golang.org/grpc"
)

type callMeta struct {
	FullMethod string
	Type       string
	Service    string
	Method     string
}

func newCallMeta(fullMethod string, streamInfo *grpc.StreamServerInfo) callMeta {
	meta := interceptors.NewServerCallMeta(fullMethod, streamInfo, nil)
	return callMeta{
		FullMethod: fullMethod,
		Type:       string(meta.Typ),
		Service:    meta.Service,
		Method:     meta.Method,
	}
}
