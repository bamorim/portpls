BINARY=portpls
MODULE=github.com/bamorim/portpls

VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT?=$(shell git rev-parse HEAD 2>/dev/null || echo "unknown")
DATE?=$(shell date -u +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || echo "unknown")
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

.PHONY: build build-dev clean fmt test install

build:
	go build $(LDFLAGS) -o bin/$(BINARY) ./cmd/portpls

build-dev:
	go build -o bin/$(BINARY) ./cmd/portpls

clean:
	rm -rf bin/

fmt:
	gofmt -w cmd internal

test:
	go test ./...

install:
	go install $(LDFLAGS) ./cmd/portpls
