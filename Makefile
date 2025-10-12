.PHONY: lint
lint:
	golangci-lint run --timeout 60s --max-same-issues 50 ./...

.PHONY: lintf
lintf:
	golangci-lint run --fix --timeout 60s --max-same-issues 50 ./...

.PHONY: build
build:
	go build ./...

.PHONY: test
test:
	go test -race -v ./...

.PHONY: test-cov
test-cov:
	mkdir -p coverage \
	&& go test -race -v ./... -coverprofile=coverage/cover.out \
	&& go tool cover -html=coverage/cover.out -o coverage/cover.html

.PHONY: bench
bench:
	go test -bench=. -benchmem .

.PHONY: all
all: lint build test-cov bench

.DEFAULT_GOAL := all
