# Quality Gates

Concrete tooling and `make` targets that enforce
[code-standards.md](code-standards.md) and [test-standards.md](test-standards.md).

---

## Prerequisites

Only Go is required. All tool dependencies are pinned in `go.mod` via the
`tool` directive and resolved automatically by `go tool`.

## Targets

| Target | Command | Purpose |
|--------|---------|---------|
| `make fmt` | `gofmt -w .` + `go tool gci write ...` | Format code and sort imports |
| `make lint` | `go tool golangci-lint run` | Static analysis (see `.golangci.yml`) |
| `make test` | `go test ./...` | Unit tests |
| `make test-integration` | `go test -tags integration ./...` | Integration tests (may need Docker) |
| `make build` | `go build -o bin/havn ./cmd/havn` | Compile binary to `bin/` |
| `make install` | `go install ./cmd/havn/` | Install to `$GOBIN` / `$GOPATH/bin` |
| `make check` | fmt + lint + test + build | Full quality gate |
| `make clean` | `rm -rf bin/` | Remove build artifacts |

## Workflow

### Local development

Run `make check` before committing. It runs all gates in order:
format, lint, test, build.

### CI

All gates run on every push. A failure on any gate blocks merge.

Integration tests run separately and may be gated on Docker availability.

## Tool versions

Tool versions are pinned in `go.mod` under the `tool` directive:

```
tool (
    github.com/daixiang0/gci
    github.com/golangci/golangci-lint/v2/cmd/golangci-lint
)
```

Update with `go get -tool <package>@latest`.

## Linter configuration

See `.golangci.yml`. The linter set and rationale are documented in
[code-standards.md](code-standards.md) Section 6.
