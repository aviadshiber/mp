BINARY_NAME=mp
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

.PHONY: build test install clean lint fmt completions release

build:
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/mp

test:
	go test -v ./...

install:
	go install $(LDFLAGS) ./cmd/mp

clean:
	rm -rf bin/ dist/

lint:
	golangci-lint run

fmt:
	go fmt ./...

completions:
	mkdir -p completions
	go run ./cmd/mp completion bash > completions/mp.bash
	go run ./cmd/mp completion zsh > completions/_mp
	go run ./cmd/mp completion fish > completions/mp.fish

release:
	goreleaser release --snapshot --clean
