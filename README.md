# grpc-api-deprecation

[![tag](https://img.shields.io/github/tag/belo4ya/grpc-api-deprecation.svg)](https://github.com/belo4ya/grpc-api-deprecation/releases)
![go version](https://img.shields.io/badge/-%E2%89%A51.24-%23027d9c?logo=go&logoColor=white&labelColor=%23555)
[![go doc](https://godoc.org/github.com/belo4ya/grpc-api-deprecation?status.svg)](https://pkg.go.dev/github.com/belo4ya/grpc-api-deprecation)
[![go report](https://goreportcard.com/badge/github.com/belo4ya/grpc-api-deprecation)](https://goreportcard.com/report/github.com/belo4ya/grpc-api-deprecation)
[![codecov](https://codecov.io/gh/belo4ya/grpc-api-deprecation/graph/badge.svg?token=GQZRP94G21)](https://codecov.io/gh/belo4ya/grpc-api-deprecation)
[![license](https://img.shields.io/github/license/belo4ya/grpc-api-deprecation)](./LICENSE)

🎯 The goal of the project TODO...

## 🚀 Install

```sh
go get -u github.com/belo4ya/grpc-api-deprecation
```

**Compatibility:** Go ≥ 1.24

## 💡 Usage

- Tiny examples in godoc: [pkg.go.dev/github.com/belo4ya/grpc-api-deprecation#pkg-examples](https://pkg.go.dev/github.com/belo4ya/grpc-api-deprecation#pkg-examples)

TODO...

## 📚 Acknowledgments

The following projects had a particular impact on the design of `grpc-api-deprecation`.

- [grpc-ecosystem/go-grpc-middleware](https://github.com/grpc-ecosystem/go-grpc-middleware) - Golang gRPC Middlewares: interceptor chaining, auth, logging, retries and more.
- [bufbuild/protovalidate-go](https://github.com/bufbuild/protovalidate-go) - Protocol Buffer Validation for Go.

---------------------------------------------

OUTDATED README PART:

gRPC unary interceptor that records Prometheus metrics when API consumers use **deprecated** fields or **deprecated enum values** defined in your protobufs. Fast, allocation-friendly after warm-up, and built around a plan/evaluator architecture inspired by protovalidate.

## What it does

* Detects use of fields annotated with `deprecated = true` (on **field options**). Emits a single counter increment per populated field.
* Detects use of **enum values** annotated with `deprecated = true` (on **enum value options**) across scalar, repeated, and map fields. Emits one increment per occurrence.
* Traverses nested messages, repeated elements, and maps with a copy-on-write, descriptor-driven plan built once per message type (no per-request reflection graph building).
* Generates human-readable field paths (e.g. `lists.messages[].field_deprecated`; collection suffixes `[]`/`{}` are trimmed on the last segment for cleaner output).
* Applies an iteration cap to huge collections to protect latency (with a separate counter to show when the cap is hit).

## Metrics

Exposed (via promauto) counters:

* `grpc_deprecated_field_used_total{grpc_service,grpc_method,field,field_presence}` — deprecated **field** was populated. `field_presence` is `explicit` (has presence, e.g., messages/oneofs/wrappers/optional) or `implicit` (proto3 scalars).
* `grpc_deprecated_enum_used_total{grpc_service,grpc_method,field,enum_value,enum_number}` — deprecated **enum value** was used.
* `grpc_deprecated_field_usage_hit_max_items_per_collection_total{grpc_service,grpc_method,field,collection_type,max_items}` — iteration over a list/map was cut due to the safety cap.

## Install

```bash
go get github.com/belo4ya/grpc-api-deprecation
```

## Quick start

Register the unary interceptor in your gRPC server:

```go
import (
  "google.golang.org/grpc"
  "github.com/belo4ya/grpc-api-deprecation"
)

srv := grpc.NewServer(
  grpc.UnaryInterceptor(apideprecation.UnaryServerInterceptor()),
)
```

Expose your Prometheus endpoint as usual (e.g., with `promhttp.Handler()`).

## Field paths & presence

* Paths use dot notation with collection hints on non-terminal segments, e.g. `maps.messages{}.field_deprecated` or `lists.messages[].field_deprecated`. The last segment is printed cleanly (`[]`/`{}` trimmed) for readability.
* `field_presence`:

    * `explicit` — fields with presence semantics (`HasPresence==true`: messages, oneofs, `optional` scalars, wrappers).
    * `implicit` — populated proto3 scalar fields without explicit presence.

## Performance notes

* Plans are built once per `MessageDescriptor` and cached with copy-on-write; subsequent requests only traverse values using the prebuilt evaluators.
* Large lists/maps are scanned up to a fixed cap (`maxItemsPerCollection`, default 50). When exceeded, scanning stops and a counter is incremented.

## API surface

* `apideprecation.UnaryServerInterceptor()` — returns a `grpc.UnaryServerInterceptor`. Plug it into your server; it introspects requests and emits metrics.

## How it works (under the hood)

* A `planBuilder` walks message descriptors and emits a set of evaluators (“nodes”) for:

    * deprecated fields (`fieldNode`),
    * nested messages/collections (`messageNode`, `listNode`, `mapNode`),
    * deprecated enum values (`enumNode`), which works uniformly for scalar, list, and map elements.
* At runtime, the plan is executed against the incoming message; evaluators push/pop a lightweight field-path and trigger the metric callbacks.

## Limitations / notes

* Only unary RPCs are covered (this interceptor is unary-only by design).
* The collection iteration cap is a constant; adjust it in code if your payloads are very large.

## Development

* Run tests & benchmarks:

```bash
go test ./...
go test -bench=. -benchmem ./...
```
