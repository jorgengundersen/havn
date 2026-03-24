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

## Pre-commit hook

Git hooks live in `.beads/hooks/` (via `core.hooksPath`). The
pre-commit hook chains two tools:

1. **Lefthook** — runs quality gates in parallel (fmt, lint, test,
   build) as defined in `lefthook.yml`.
2. **Beads** — JSONL sync and issue tracking integration, injected
   via `bd hooks install --beads --chain` using section markers.

Lefthook owns the hook file. Beads appends its section between
`BEGIN/END BEADS INTEGRATION` markers, which lefthook preserves
across syncs. To reinstall after changes:

```bash
lefthook install --force    # install lefthook hook shim
bd hooks install --beads --chain  # chain beads into it
```

A failure in any gate blocks the commit.

## Workflow

### Local development

`make check` runs automatically via pre-commit hook. Run it manually
to verify before staging.

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
