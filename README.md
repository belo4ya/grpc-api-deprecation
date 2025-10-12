# grpc-api-deprecation

[![tag](https://img.shields.io/github/tag/belo4ya/grpc-api-deprecation.svg)](https://github.com/belo4ya/grpc-api-deprecation/releases)
![go version](https://img.shields.io/badge/-%E2%89%A51.24-%23027d9c?logo=go&logoColor=white&labelColor=%23555)
[![go doc](https://godoc.org/github.com/belo4ya/grpc-api-deprecation?status.svg)](https://pkg.go.dev/github.com/belo4ya/grpc-api-deprecation)
[![go report](https://goreportcard.com/badge/github.com/belo4ya/grpc-api-deprecation)](https://goreportcard.com/report/github.com/belo4ya/grpc-api-deprecation)
[![codecov](https://codecov.io/gh/belo4ya/grpc-api-deprecation/graph/badge.svg?token=GQZRP94G21)](https://codecov.io/gh/belo4ya/grpc-api-deprecation)
[![license](https://img.shields.io/github/license/belo4ya/grpc-api-deprecation)](./LICENSE)

🎯 The goal of the project TODO...

## ✨ Features

- 🔍 Detects `deprecated = true` flags on services, methods, fields, and enum
  values straight from protobuf descriptors while emitting four Prometheus
  counters with the base labels `grpc_type`, `grpc_service`, and `grpc_method`:
  `grpc_deprecated_method_used_total`, `grpc_deprecated_field_used_total`,
  `grpc_deprecated_enum_used_total`, and
  `grpc_deprecated_field_usage_hit_max_items_per_collection_total` (enriched with
  field paths, presence, enum value, and collection limits where relevant).
- 🧰 Offers extensive customization: add per-metric dynamic labels and
  exemplars (`WithExtraLabels`, `WithExemplar`), prewarm caches for known
  services (`WithPrewarm`), and tweak counter construction (`WithCounterOptions`).
  See [options.go](./options.go) for the full list of tuning knobs.
- ⚡ Prioritizes throughput with lock-free hot paths, evaluator reuse, and
  descriptor caching — see [Performance](#-performance) for benchmark numbers and
  optimization details.

Three counters are exported; all use the base labels `grpc_type`, `grpc_service`, and `grpc_method`:

| Name                                                             | Additional labels                                           | Description                                                                                    |
|------------------------------------------------------------------|-------------------------------------------------------------|------------------------------------------------------------------------------------------------|
| `grpc_deprecated_method_used_total`                              | *(defaults + extra method labels)*                          | Deprecated RPC method was invoked (method or service marked deprecated).                       |
| `grpc_deprecated_field_used_total`                               | `field`, `field_presence`, *(extra field labels)*           | Deprecated field with `deprecated = true` was populated. Presence is `explicit` or `implicit`. |
| `grpc_deprecated_enum_used_total`                                | `field`, `enum_value`, `enum_number`, *(extra enum labels)* | Deprecated enum value was observed in the request payload.                                     |
| `grpc_deprecated_field_usage_hit_max_items_per_collection_total` | `field`, `collection_type`, `max_items`                     | Scanner bailed out after hitting the collection safety cap.                                    |

## 🚀 Install

```sh
go get -u github.com/belo4ya/grpc-api-deprecation
```

**Compatibility:** Go ≥ 1.24

## 💡 Usage

Register the metrics collector, hook the interceptors into your gRPC server, and
expose Prometheus metrics as usual:

```go
import (
    apideprecation "github.com/belo4ya/grpc-api-deprecation"
    "github.com/prometheus/client_golang/prometheus"
    "google.golang.org/grpc"
)

metrics := apideprecation.NewMetrics()
prometheus.MustRegister(metrics)

srv := grpc.NewServer(
    grpc.ChainUnaryInterceptor(metrics.UnaryServerInterceptor()),
    grpc.ChainStreamInterceptor(metrics.StreamServerInterceptor()),
)
```

Plug `srv` into your existing `promhttp.Handler()` (or any other exporter) to make the counters available to Prometheus.

## 🏎️ Performance

`grpc-api-deprecation` keeps the hot path lean by caching descriptor lookups,
reusing evaluation plans with copy-on-write updates, and minimizing allocations
while walking nested messages, maps, and repeated fields. Cold starts build the
necessary plan only once per method and reuse it afterwards.

```
BenchmarkUnaryServerInterceptor/XS-12              6819776             171.3 ns/op             320 B/op          2 allocs/op
BenchmarkUnaryServerInterceptor/S-12               1000000              1190 ns/op             744 B/op         10 allocs/op
BenchmarkUnaryServerInterceptor/M-12                199286              6072 ns/op            2738 B/op         41 allocs/op
BenchmarkUnaryServerInterceptor/L-12                 43059             25760 ns/op            9770 B/op        142 allocs/op
BenchmarkUnaryServerInterceptor/cold_start/XS-12    354134              3596 ns/op            4012 B/op         75 allocs/op
BenchmarkUnaryServerInterceptor/cold_start/S-12     147796              8217 ns/op            8720 B/op        186 allocs/op
BenchmarkUnaryServerInterceptor/cold_start/M-12      37108             29228 ns/op           28076 B/op        575 allocs/op
BenchmarkUnaryServerInterceptor/cold_start/L-12       9810            115663 ns/op          105888 B/op       1981 allocs/op
```

For the best latencies in production, pre-populate caches using
`WithPrewarm(...grpc.ServiceDesc)` before serving traffic. If you register custom
label or exemplar extractors, remember their value resolvers execute on every
matching request — any extra CPU or allocation costs introduced by those
functions are your responsibility.

## 📚 Acknowledgments

The following projects influenced `grpc-api-deprecation`:

- [grpc-ecosystem/go-grpc-middleware](https://github.com/grpc-ecosystem/go-grpc-middleware)
- [bufbuild/protovalidate-go](https://github.com/bufbuild/protovalidate-go)

## 🔜 TODO

-
-
-
-
-
-
-
