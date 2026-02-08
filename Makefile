SHELL := /bin/bash
PATH := $(GOPATH)/bin:$(PATH)
BINARY := caster-generator

.PHONY: all build clean lint test cover bench help

## help: Show this help message
help:
	@echo "caster-generator Makefile"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^## ' Makefile | sed 's/^## /  /'
	@echo ""
	@echo "Examples:"
	@echo "  make build    # Build the binary"
	@echo "  make test     # Run tests"
	@echo "  make all      # Run clean, lint, test, bench"

## build: Build the caster-generator binary
build:
	go build -o $(BINARY) ./cmd/caster-generator/

## lint: Run golangci-lint with auto-fix (excludes examples/)
lint:
	@DIRS=$$(ls -la | awk '/^d/ && $$NF !~ /^\./ && $$NF !~ /examples/ {print "./" $$NF "/..."}'); \
	golangci-lint run --fix $$DIRS

## test: Run all tests with race detection
test:
	go test -race ./internal/...

## cover: Run tests with coverage and open HTML report
cover:
	go test -coverprofile=coverage.out ./internal/...
	go tool cover -html=coverage.out

## bench: Run benchmarks
bench:
	go test -bench=. ./internal/...

## clean: Clean test cache and built binary
clean:
	go clean -testcache
	rm -f $(BINARY) coverage.out

## all: Run clean, lint, test, bench, build
all: clean lint test bench build
