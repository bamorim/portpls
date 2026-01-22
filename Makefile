BINARY=portpls

.PHONY: build fmt test

build:
	go build -o bin/$(BINARY) ./cmd/portpls

fmt:
	gofmt -w cmd/portpls internal


test:
	go test ./...
