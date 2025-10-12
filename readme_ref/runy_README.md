# runy - tiny run manager

[![tag](https://img.shields.io/github/tag/belo4ya/runy.svg)](https://github.com/belo4ya/runy/releases)
![go version](https://img.shields.io/badge/-%E2%89%A51.20-%23027d9c?logo=go&logoColor=white&labelColor=%23555)
[![go doc](https://godoc.org/github.com/belo4ya/runy?status.svg)](https://pkg.go.dev/github.com/belo4ya/runy)
[![go report](https://goreportcard.com/badge/github.com/belo4ya/runy)](https://goreportcard.com/report/github.com/belo4ya/runy)
[![codecov](https://codecov.io/gh/belo4ya/runy/graph/badge.svg?token=GQZRP94G21)](https://codecov.io/gh/belo4ya/runy)
[![license](https://img.shields.io/github/license/belo4ya/runy)](./LICENSE)

🎯 The goal of the project is to provide developers with the opportunity not to think about the graceful shutdown
and not to make mistakes in its implementation in their Go application.
Instead, focus on startup components such as HTTP and gRPC servers and other `Runnable`s.

## 🚀 Install

```sh
go get -u github.com/belo4ya/runy
```

**Compatibility:** Go ≥ 1.20

## 💡 Usage

- Tiny examples in godoc: [pkg.go.dev/github.com/belo4ya/runy#pkg-examples](https://pkg.go.dev/github.com/belo4ya/runy#pkg-examples)
- End-to-end usage examples: [examples/](examples)
- `Runnable` recipes (http, grpc servers, kafka consumers): [examples/runnables](examples/runnables)

You can import `runy` using:

```go
import (
	"github.com/belo4ya/runy"
)
```

Then implement the `Runnable` interface for your application components:

```go
// Runnable allows a component to be started.
// It's very important that Start blocks until it's done running.
type Runnable interface {
	// Start starts running the component.
	// The component will stop running when the context is closed.
	// Start blocks until the context is closed or an error occurs.
	Start(ctx context.Context) error
}
```

Finally, register and run your components using `runy.Add` and `runy.Start`.
Here's a simple application with multiple HTTP servers:

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/belo4ya/runy"
)

func main() {
	// Create a context that's canceled when SIGINT or SIGTERM is received.
	ctx := runy.SetupSignalHandler()

	// Initialize and register application components with runy.
	httpSrv := NewHTTPServer(":8080") // main API server
	mgmtSrv := NewHTTPServer(":8081") // /metrics, /debug/pprof, /healthz, /readyz
	runy.Add(httpSrv, mgmtSrv)

	// Start all components and block until shutdown.
	log.Println("starting app")
	if err := runy.Start(ctx); err != nil {
		log.Fatalf("app error: %v", err)
	}
}

type HTTPServer struct {
	HTTP *http.Server
}

func NewHTTPServer(addr string) *HTTPServer {
	return &HTTPServer{HTTP: &http.Server{Addr: addr}}
}

// Start implements Runnable interface.
// It starts the HTTP server and blocks until context cancellation or server error.
func (s *HTTPServer) Start(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		log.Printf("http server starts listening on: %s", s.HTTP.Addr)
		if err := s.HTTP.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("http listen and serve: %w", err)
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		log.Println("shutting down http server")
		// Handle cleanup after context cancellation with a reasonable shutdown timeout.
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := s.HTTP.Shutdown(ctx); err != nil {
			return fmt.Errorf("http shutdown: %w", err)
		}
		return nil
	case err := <-errCh:
		return err
	}
}
```

## 🧠 Core Concepts

Runnable, SugaredRunnable, Group...

## 📚 Acknowledgments

The following projects had a particular impact on the design of `runy`.

- [kubernetes-sigs/controller-runtime](https://github.com/kubernetes-sigs/controller-runtime) - set of go libraries for building Kubernetes controllers.
- [oklog/run](https://github.com/oklog/run) - universal mechanism to manage goroutine lifecycles.
- [go-kratos/kratos](https://github.com/go-kratos/kratos) - ultimate Go microservices framework for the cloud-native era.
- [sourcegraph/conc](https://github.com/sourcegraph/conc) - better structured concurrency for Go.
