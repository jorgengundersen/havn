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
	go install $(LDFLAGS) ./cmd/havn/

check: fmt-check lint test-contract-matrix test build

clean:
	rm -rf bin/
