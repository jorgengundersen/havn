.PHONY: build test test-integration test-boundary-confidence lint fmt fmt-check check install clean

build:
	go build -o bin/havn ./cmd/havn

test:
	go test ./...

test-integration:
	go test -tags integration ./...

test-boundary-confidence:
	go test ./internal/cli -run 'TestHAVNBinary_CLIContractAtProcessBoundary|TestDoctorCommand_'
	go test -tags integration ./internal/dolt -run TestSharedDoltLifecycleAndReadiness_Integration

lint:
	go tool golangci-lint run

fmt:
	gofmt -w .
	go tool gci write --section standard --section default --section "prefix(github.com/jorgengundersen/havn)" .

fmt-check:
	@out="$$(gofmt -l .)"; [ -z "$$out" ] || { echo "gofmt check failed - run 'make fmt'"; exit 1; }
	@out="$$(go tool gci list --section standard --section default --section 'prefix(github.com/jorgengundersen/havn)' .)"; [ -z "$$out" ] || { echo "gci import order check failed - run 'make fmt'"; exit 1; }

install:
	go install ./cmd/havn/

check: fmt-check lint test build

clean:
	rm -rf bin/
