.PHONY: build test lint fmt check install clean

build:
	go build -o bin/havn ./cmd/havn

test:
	go test ./...

test-integration:
	go test -tags integration ./...

lint:
	go tool golangci-lint run

fmt:
	gofmt -w .
	go tool gci write --section standard --section default --section "prefix(github.com/jorgengundersen/havn)" .

install:
	go install ./cmd/havn/

check: fmt lint test build

clean:
	rm -rf bin/
