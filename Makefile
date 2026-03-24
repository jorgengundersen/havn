.PHONY: build test lint fmt check clean

build:
	go build ./...

test:
	go test ./...

test-integration:
	go test -tags integration ./...

lint:
	golangci-lint run

fmt:
	gofmt -w .
	gci write --section standard --section default --section "prefix(github.com/jorgengundersen/havn)" .

check: fmt lint test build

clean:
	rm -rf bin/
