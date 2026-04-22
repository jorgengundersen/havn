# Quality Gates

Concrete tooling and `make` targets that enforce
[code-standards.md](code-standards.md) and [test-standards.md](test-standards.md).

---

## Prerequisites

Go is required. Static analysis uses `staticcheck`, pinned in `go.mod` via the
`tool` directive and executed through `go tool`.

## Targets

| Target | Command | Purpose |
|--------|---------|---------|
| `make fmt` | `gofmt -w .` | Format code |
| `make fmt-check` | `gofmt -l .` | Validate formatting without rewriting files |
| `make lint` | `go vet ./...` + `go tool staticcheck ./...` | Lean static analysis focused on correctness |
| `make test` | `go test ./...` | Unit tests |
| `make test-contract-matrix` | Contract scenarios in `internal/container` and `internal/cli` | Authoritative environment-interface matrix gate |
| `make test-integration` | `go test -tags integration ./...` | Integration tests (may need Docker) |
| `make test-boundary-confidence` | CLI binary contract + doctor CLI behavior + shared Dolt integration subset | Boundary-confidence suites for shipped behavior |
| `make build` | `go build -o bin/havn ./cmd/havn` | Compile binary to `bin/` |
| `make install` | `go install ./cmd/havn/` | Install to `$GOBIN` / `$GOPATH/bin` |
| `make check` | fmt-check + lint + test-contract-matrix + test + build | Full quality gate on the committed tree |
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
2. **Beads** — runs `bd hooks run pre-commit` (Dolt sync)

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
| `pre-commit` | Quality gates | Dolt sync |
| `prepare-commit-msg` | — | Agent identity trailers |
| `post-merge` | — | Dolt sync after pull |
| `post-checkout` | — | Dolt sync after checkout |
| `pre-push` | — | Validation before push |

## Workflow

### Local development

Lefthook runs the quality gate jobs automatically on commit when `.go`
files are staged. Jobs run in parallel for speed. Use `make check` to
run all gates manually (sequential, always runs regardless of file
types).

### CI

CI runs in GitHub Actions.

The core quality-gate job should invoke `make check` so CI and local
development share the same contract.

All core gates run on every push and pull request. A failure on any core gate
blocks merge.

Integration tests run in a separate job via `make test-integration` and may be
gated on Docker-capable runners.

Integration failures should be visible as a separate CI result rather than
folded into the core quality-gate job.

Boundary-confidence suites should run in a dedicated CI job via
`make test-boundary-confidence` so shipped CLI boundary contracts, doctor
effective-state behavior, and shared-Dolt readiness paths are continuously
validated as an explicit merge signal.

`quality-gates`, `integration-tests`, and `boundary-confidence` are required merge checks for `main`, matching `.github/settings.yml` branch protection.
At minimum, `integration-tests` and `boundary-confidence` are required merge checks.

## Toolchain dependency-surface decision

The quality-gate standard is intentionally lean: `gofmt`, `go vet`,
`staticcheck`, `go test`, and `go build`.

This keeps dependency surface low while preserving high-signal correctness
checks for a Go CLI with container/runtime integrations.

The project does not require a multi-linter orchestrator for merge gates.
If additional checks are proposed, they must justify their signal and
maintenance cost, not only stylistic preference.

## Tool versions

Tool versions are pinned in `go.mod` under the `tool` directive:

```
tool (
    honnef.co/go/tools/cmd/staticcheck
)
```

Update with `go get -tool <package>@latest`.

## Linter configuration

Lint behavior is defined by `go vet` and `staticcheck` invocation in the
quality-gate commands above. Rationale is documented in
[code-standards.md](code-standards.md) Section 6.
