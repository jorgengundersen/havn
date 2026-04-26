.PHONY: build test test-contract-matrix test-integration test-boundary-confidence lint fmt fmt-check check install clean

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -ldflags "-X github.com/jorgengundersen/havn/internal/cli.version=$(VERSION)"

build:
	go build $(LDFLAGS) -o bin/havn ./cmd/havn

test:
	go test ./...

test-contract-matrix:
	@echo "Running environment-interface contract matrix tests"
	go test ./internal/container -run 'TestStartOrAttach_RunningContainer_(PrepareCapabilityFailure_AbortsWithGuidance|MissingOptionalPrepareCapability_Continues|MissingRequiredDevShell_AbortsBeforePrepare)'
	go test ./internal/cli -run TestStartupContractMatrix_RootAndUpAndEnter

test-integration:
	go test -tags integration ./...

test-boundary-confidence:
	go run ./internal/ci/cmd/boundaryconfidence

lint:
	go vet ./...
	go tool staticcheck ./...

fmt:
	gofmt -w .

fmt-check:
	@out="$$(gofmt -l .)"; [ -z "$$out" ] || { echo "gofmt check failed - run 'make fmt'"; exit 1; }

install:
	go install $(LDFLAGS) ./cmd/havn/

check: fmt-check lint test-contract-matrix test build

clean:
	rm -rf bin/
