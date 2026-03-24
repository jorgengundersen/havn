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

## Git hooks

Git hooks live in `.beads/hooks/` (via `core.hooksPath`). Two tools
share the hook files with distinct responsibilities:

| Tool | Role | Owns hook file? | Config |
|------|------|-----------------|--------|
| **Lefthook** | Quality gates (fmt, lint, test, build) | Yes — writes the hook shim | `lefthook.yml` |
| **Beads** | Issue tracking sync, commit metadata | No — appends via section markers | `.beads/config.yaml` |

### How they coexist

Lefthook owns the hook files and writes its shim (the script that
finds and invokes `lefthook run`). Beads injects its integration
between `BEGIN/END BEADS INTEGRATION` markers using the `--chain`
flag. Lefthook preserves content outside its own shim across syncs,
so the beads section survives `lefthook run` and `lefthook install`.

The pre-commit hook executes in this order:
1. **Lefthook** — runs quality gate jobs in parallel (defined in
   `lefthook.yml`)
2. **Beads** — runs `bd hooks run pre-commit` (JSONL sync)

A failure in either tool blocks the commit.

### Hook setup

After cloning or when hooks need reinstalling:

```bash
lefthook install --force          # 1. install lefthook hook shim
bd hooks install --beads --chain  # 2. chain beads into it
```

Order matters — lefthook must be installed first so beads can chain
into the existing hook file. Running `bd hooks install` without
`--chain` would replace the lefthook shim.

### All managed hooks

| Hook | Lefthook | Beads |
|------|----------|-------|
| `pre-commit` | Quality gates | JSONL sync |
| `prepare-commit-msg` | — | Agent identity trailers |
| `post-merge` | — | JSONL import after pull |
| `post-checkout` | — | JSONL import after checkout |
| `pre-push` | — | Validation before push |

## Workflow

### Local development

Lefthook runs the quality gate jobs automatically on commit when `.go`
files are staged. Jobs run in parallel for speed. Use `make check` to
run all gates manually (sequential, always runs regardless of file
types).

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
