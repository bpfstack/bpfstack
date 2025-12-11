PROJECT_DIR := $(shell pwd)

.PHONY: all vmlinux generate build-agent run-agent clean lint test

all: build-agent

vmlinux:
	@echo "Generating vmlinux.h"
	mkdir -p headers
	bpftool btf dump file /sys/kernel/btf/vmlinux format c > headers/vmlinux.h

generate: vmlinux
	@echo "Generating bpf programs"
	HEADER_DIR=$(PROJECT_DIR)/headers go generate ./...

build-agent: generate
	@echo "Building agent"
	go build -o bin/agent ./cmd/agent

run-agent: build-agent
	@echo "Running agent"
	sudo ./bin/agent

lint:
	@echo "Running golangci-lint"
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "golangci-lint not found. Installing..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.60.0; \
	fi
	golangci-lint run

test:
	@echo "Running tests"
	go test -v -race -coverprofile=coverage.out ./...

test-coverage: test
	@echo "Generating coverage report"
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

clean:
	@echo "Cleaning build artifacts"
	rm -rf bin headers/vmlinux.h coverage.out coverage.html
	find . -name "*_bpfel_*.go" -delete
	find . -name "*.o" -delete