PROJECT_DIR := $(shell pwd)

.PHONY: all vmlinux generate build-agent run-agent clean

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

clean:
	@echo "Cleaning build artifacts"
	rm -rf bin headers/vmlinux.h
	find . -name "*_bpfel_*.go" -delete
	find . -name "*.o" -delete