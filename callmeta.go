package apideprecation

import (
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors"
	"google.golang.org/grpc"
)

type CallMeta struct {
	FullMethod string
	Type       string
	Service    string
	Method     string
}

func newCallMeta(fullMethod string, streamInfo *grpc.StreamServerInfo) CallMeta {
	meta := interceptors.NewServerCallMeta(fullMethod, streamInfo, nil)
	return CallMeta{
		FullMethod: fullMethod,
		Type:       string(meta.Typ),
		Service:    meta.Service,
		Method:     meta.Method,
	}
}
