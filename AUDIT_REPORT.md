# Audit Report

Systematic audit of havn's current state. Each section corresponds to a
child task of the parent epic (`havn-qf6`).

---

## Documentation and Onboarding

_Audited: 2026-04-11 | Issue: havn-qf6.6_

### What Exists

| File | Size | Content |
|------|------|---------|
| `README.md` | 6 bytes | Title only (`# havn`) — no description, install instructions, usage, or examples |
| `CLAUDE.md` | 11 bytes | Single `@AGENTS.md` directive — delegates entirely to AGENTS.md |
| `AGENTS.md` | ~5 KB | Comprehensive agent instructions: bd workflow, non-interactive shell conventions, session completion checklist, memory policy |
| `LICENSE` | 1 KB | MIT license present |
| `specs/README.md` | ~400 bytes | Clean index table linking all 9 spec files with one-line descriptions |
| `specs/*.md` (×8) | Varies | Architecture principles, code standards, test standards, quality gates, CLI framework, havn overview, havn doctor, shared Dolt server |
| `Makefile` | 408 bytes | Present (build tooling) |
| `PROMPT.md` | 888 bytes | Headless agent runner prompt |

### What's Missing

| Gap | Impact |
|-----|--------|
| **No user-facing README content** | A new user or contributor visiting the repo sees only `# havn`. No project description, installation instructions, usage examples, prerequisites, or quickstart. This is the single biggest onboarding blocker. |
| **No `docs/` directory** | No user documentation beyond specs. No install guide, no usage guide, no architecture overview for humans. Specs are excellent engineering references but assume deep context. |
| **No `CONTRIBUTING.md`** | No contributor onboarding: no dev setup instructions, no PR workflow, no coding guidelines summary. Contributors must reverse-engineer the workflow from AGENTS.md and specs. |
| **No `CHANGELOG.md`** | No release history. Minor for pre-1.0, but becomes important at first public release. |
| **No Go doc comments on exported types** | `go doc` produces minimal output. The code is internal-only, so impact is limited to developer navigation. |
| **CLAUDE.md is a single redirect** | Works fine for agents that resolve `@AGENTS.md`, but any tool that reads CLAUDE.md literally sees no instructions. Not a blocker in practice (Claude Code resolves it), but fragile if other tools consume the file. |

### Spec Discoverability

**Good:** `specs/README.md` provides a well-organized index table. All 8 non-index specs are listed with descriptions. Navigation between specs uses relative links that work correctly.

**Gap:** The project root has no pointer to `specs/`. A new contributor would need to know to look in that directory. README.md should link to `specs/README.md` as the technical reference.

### Agent Onboarding (AGENTS.md)

**Strengths:**
- Clear bd workflow with examples
- Non-interactive shell conventions (prevents agent hangs)
- Session completion checklist (ensures push)
- Memory policy (bd over file-based for havn-specific context)

**Gaps:**
- No mention of how to run the project (`go build`, `make`, test commands)
- No mention of project structure or where to find code (`cmd/`, `internal/`)
- References `docs/QUICKSTART.md` which does not exist
- Relies on `bd onboard` for context, but doesn't describe what that provides

### Impact Assessment

| Audience | Onboarding Quality |
|----------|--------------------|
| **AI agents (Claude Code)** | Adequate — AGENTS.md + specs + bd provide enough context for structured work. Gaps: stale `docs/QUICKSTART.md` reference, no project structure overview. |
| **Human contributors** | Poor — README is empty, no CONTRIBUTING.md, no dev setup guide. Must discover specs/ independently and read multiple files to understand the project. |
| **End users** | Non-existent — no installation, usage, or configuration documentation. Users cannot learn what havn does or how to use it from the repo alone. |

### Recommendations (not actioned — audit only)

1. **README.md** — Add: project description, prerequisites, installation, basic usage, link to specs/
2. **CONTRIBUTING.md** — Dev setup, make targets, test commands, PR workflow, link to code-standards.md
3. **AGENTS.md** — Remove stale `docs/QUICKSTART.md` reference; add project structure overview
4. **User docs** — At minimum, a usage guide covering `havn .`, `havn stop`, configuration
5. **CHANGELOG.md** — Start tracking before first public release

---

## Code Quality and Architecture

_Audited: 2026-04-11 | Issue: havn-qf6.4_

Assessed against specs/code-standards.md §1–§7 and specs/architecture-principles.md §1–§12.

### Package Structure (code-standards §1, principles §1, §7, §11)

| Criterion | Status | Evidence |
|-----------|--------|----------|
| Entry point in `cmd/havn/` | ✅ MET | `cmd/havn/main.go` — wiring only, calls `cli.Execute()` |
| Everything else under `internal/` | ✅ MET | 10 packages: `cli`, `config`, `container`, `docker`, `doctor`, `dolt`, `mount`, `name`, `volume` |
| Domain-first package names | ✅ MET | No `utils/`, `helpers/`, `common/`, `types/`, `models/` |
| One concern per package | ✅ MET | Each package has a single domain responsibility |
| No `pkg/` directory | ✅ MET | Everything is internal |
| No circular dependencies | ✅ MET | Import graph is a strict DAG |

**Import layering:**

```
cmd/havn → cli → {config, docker, doctor, dolt, volume}
                   ↓
           domain packages: container, mount, name, volume, dolt, doctor
                   ↓
           infrastructure: docker (wraps Docker SDK)
```

Domain packages import only stdlib, `config`, `mount`, and `name`. Docker SDK types never appear outside `internal/docker/`.

**Import ordering** enforced by `gci` in `.golangci.yml`: stdlib → third-party → internal. Consistent across all files.

### Dependency Isolation (code-standards §4, principles §4, §12)

| Criterion | Status | Evidence |
|-----------|--------|----------|
| Interfaces defined by consumer | ✅ MET | 14 interfaces across `container/`, `dolt/`, `volume/`, `doctor/`, `cli/` — all consumer-defined with explicit `// Consumer-defined per code-standards §4` comments |
| Compile-time assertions | 🔴 MISSING | No `var _ Interface = (*Type)(nil)` in `internal/docker/` |
| Constructor injection | ✅ MET | 24 `New*` constructors; all composed units receive deps as params |
| Wrapper hides external client | ✅ MET | `docker.Client.docker` is unexported `*client.Client` |
| Boundary translation | ✅ MET | All Docker SDK errors converted to domain errors at wrapper |

**Consumer-defined interfaces (14 total):**

- `container.Backend`, `container.StartBackend`, `container.NetworkBackend`, `container.VolumeEnsurer`, `container.MountResolver`, `container.DoltSetup`, `container.ExecBackend`, `container.ImageBackend`, `container.StopBackend` — in `internal/container/`
- `dolt.Backend` — in `internal/dolt/backend.go`
- `volume.Backend` — in `internal/volume/backend.go`
- `doctor.Backend` — in `internal/doctor/backend.go`
- `doctor.Check` — in `internal/doctor/check.go`
- `cli.TypedError` — in `internal/cli/errors.go`

**Gap — compile-time assertions:** The spec (§4) requires `var _ container.Runtime = (*Client)(nil)` for real implementations. `docker.Client` implements ~9 interfaces across its methods but has no assertions. This means interface drift (e.g., adding a param to `Backend.ContainerList`) would only fail at the call site, not at the declaration. Missing assertions for:

- `container.StartBackend`, `container.Backend`, `container.StopBackend`, `container.ImageBackend`, `container.NetworkBackend`, `container.VolumeEnsurer`, `container.ExecBackend` — should be in `internal/docker/container.go` or `client.go`
- `volume.Backend` — should be in `internal/docker/volume.go`
- `doctor.Backend` — should be in `internal/docker/daemon.go`

### Error Handling (code-standards §2, principles §5)

| Criterion | Status | Evidence |
|-----------|--------|----------|
| Custom error types at failure points | ✅ MET | 6 error files across `cli`, `config`, `container`, `docker`, `dolt`, `mount` |
| TypedError interface defined | ✅ MET | `internal/cli/errors.go` — `ErrorType() string` + `ErrorDetails() map[string]any` |
| TypedError implemented | ✅ MET | 8 types: `DaemonUnreachableError`, `ContainerNotFoundError`, `ImageNotFoundError`, `NetworkNotFoundError`, `VolumeNotFoundError`, `ImageBuildError`, `ParseError`, `ValidationError` |
| Error wrapping with `%w` | ✅ MET | 73 instances of `fmt.Errorf` with `%w` across 22 files |
| Boundary translation in docker wrapper | ✅ MET | `cerrdefs.IsNotFound()` → domain error types at every wrapper method |
| User-facing formatting at CLI only | ✅ MET | `cli.FormatError()` and `Output.Error()` — domain code never formats |
| No log-and-return | ✅ MET | Zero violations found |
| No panic for expected conditions | ✅ MET | No `panic()` in production code |

**Sentinel errors (2):**
- `cli.ErrNotImplemented` — stub commands
- `docker.ErrNetworkAlreadyExists` — network idempotency

**Observation:** `container.NotFoundError` and several dolt errors (`StartError`, `HealthCheckTimeoutError`, `DatabaseExistsError`, etc.) do not implement `TypedError`. Per spec, this is acceptable — "not every domain error needs TypedError." These carry minimal structured context beyond a name/message.

### Logging and Output (code-standards §5, principles §9)

| Criterion | Status | Evidence |
|-----------|--------|----------|
| `log/slog` from stdlib | ✅ MET | `internal/cli/logger.go` uses `slog` |
| Handler setup at program start | ✅ MET | `SetupLogger(verbose, jsonOutput bool) *slog.Logger` in cli package |
| Logger via DI (not globals) | ⚠️ PARTIAL | `Deps.Logger` field exists but not yet passed to domain packages |
| Stream separation (stderr/stdout) | ✅ MET | Logger → `os.Stderr`; data output → `o.stdout` |
| Standard attribute names | N/A | No logging calls in domain code yet (skeleton phase) |
| No log-and-return | ✅ MET | Zero violations |

**Detail:** The logger infrastructure is correctly built — `SetupLogger` creates the right handler, `Deps` struct holds it, `cli.Execute()` wires it. However, no domain package currently receives or uses a logger. When logging is added, constructors should accept `*slog.Logger` and use the standard attribute vocabulary (`component`, `operation`, `container_name`, etc.).

**`sloglint`** is configured in `.golangci.yml` and will enforce: `attr-only`, `no-global: all`, `static-msg`, `key-naming-case: snake` — ensuring compliance when logging calls are added.

### Type System (code-standards §3, principles §10)

| Criterion | Status | Evidence |
|-----------|--------|----------|
| Named types for distinct values | ✅ MET | `name.ContainerName`, `name.VolumeName`, `name.NetworkName` in `internal/name/types.go` |
| Config structs as plain data | ✅ MET | `config.Config` with TOML tags; `Resolve()`, `Validate()` as pure functions |
| Options structs for 3+ params | ✅ MET | `container.CreateOpts`, `docker.ExecOpts`, `container.BuildOpts`, `mount.ResolveOpts` |

### Go Idioms (code-standards §7, principles §1, §6)

| Criterion | Status | Evidence |
|-----------|--------|----------|
| `ctx context.Context` first param | ✅ MET | All 23 docker methods + domain orchestrators; pure functions correctly omit |
| Receiver names short & consistent | ✅ MET | `c` for `Client`, `m` for `Manager`, `s` for `Setup`, `r` for `Runner` |
| Table-driven tests | ✅ MET | Used across `config/`, `name/`, `container/`, `mount/` test files |
| `errgroup` for structured concurrency | ✅ MET | `container/stop.go` uses `errgroup.WithContext` for parallel stops |

### Dependency Graph Assessment

**External dependencies used in production code:**

| Dependency | Used By | Boundary |
|------------|---------|----------|
| `docker/docker` SDK | `internal/docker/` only | Wrapper — types never leak |
| `BurntSushi/toml` | `internal/config/config.go` only | Config loading boundary |
| `spf13/cobra` | `internal/cli/` only | CLI framework boundary |
| `stretchr/testify` | `*_test.go` only | Test assertions only |

All external dependencies are confined to their boundary packages. Domain packages (`container`, `dolt`, `doctor`, `mount`, `name`, `volume`) depend only on stdlib and other internal packages.

### Compliance Summary

| Area | Compliance | Gaps |
|------|-----------|------|
| Package structure | ✅ Full | — |
| Import graph | ✅ Full | — |
| Dependency isolation | ⚠️ Near-full | Missing compile-time assertions in `docker/` |
| Error handling | ✅ Full | — |
| TypedError | ✅ Full | — |
| Logging infrastructure | ⚠️ Partial | Logger DI ready but not yet injected into domains |
| Type system | ✅ Full | — |
| Go idioms | ✅ Full | — |
| External dep containment | ✅ Full | — |

### Recommendations (audit only — no code changes)

1. **Add compile-time assertions** in `internal/docker/` for all consumer-defined interfaces. This is the only spec-required pattern not yet implemented.
2. **Inject logger into domain packages** when logging calls are needed. The infrastructure is ready; constructors need a `*slog.Logger` parameter.
3. **Consider `TypedError`** for dolt errors if JSON consumers will need to distinguish dolt-specific failures programmatically.

---

## Infrastructure

_Audited: 2026-04-11 | Issue: havn-qf6.5_

### Build Tooling

**Makefile** (408 bytes, 8 targets)

| Target | Command | Purpose |
|--------|---------|---------|
| `build` | `go build -o bin/havn ./cmd/havn` | Compile binary to `bin/` |
| `test` | `go test ./...` | Run unit tests |
| `test-integration` | `go test -tags integration ./...` | Integration tests (requires Docker) |
| `lint` | `go tool golangci-lint run` | Static analysis via golangci-lint |
| `fmt` | `gofmt -w .` + `go tool gci write ...` | Format code and sort imports |
| `install` | `go install ./cmd/havn/` | Install binary to `$GOBIN` |
| `check` | fmt → lint → test → build | Full quality gate (sequential) |
| `clean` | `rm -rf bin/` | Remove build artifacts |

All targets match quality-gates.md §Targets exactly. No drift between spec and implementation.

**Lefthook** (`lefthook.yml`, 417 bytes)

Pre-commit hook runs 4 parallel jobs (only triggered when `.go` files are staged):

| Job | Command | Purpose |
|-----|---------|---------|
| `fmt` | `gofmt -l . \| grep ...` | Check formatting (fails if unformatted files found) |
| `lint` | `go tool golangci-lint run` | Linter gate |
| `test` | `go test ./...` | Unit test gate |
| `build` | `go build -o /dev/null ./cmd/havn` | Compilation check (output discarded) |

Hooks live in `.beads/hooks/` via `core.hooksPath`. Lefthook owns the hook files; beads chains into them via `BEGIN/END BEADS INTEGRATION` markers. This matches quality-gates.md §Git hooks exactly.

### Linter Configuration

`.golangci.yml` (version 2, 866 bytes)

**Correctness linters:**
- `govet` — catches shadow, printf mismatches
- `errcheck` — unchecked error returns
- `staticcheck` — comprehensive static analysis
- `unused` — dead code detection

**Consistency linters:**
- `revive` — 9 rules enabled: `blank-imports`, `exported`, `unexported-return`, `unused-parameter`, `var-naming`, `error-return`, `error-naming`, `receiver-naming`, `indent-error-flow`
- `sloglint` — enforces structured slog usage: `no-mixed-args`, `attr-only`, `no-global: all`, `static-msg`, `key-naming-case: snake`

**Formatter:**
- `gci` — import ordering: stdlib → third-party → internal (`github.com/jorgengundersen/havn`)

This matches code-standards.md §6 exactly. The `sloglint` configuration enforces the slog conventions from code-standards.md §5. `revive` rules align with the naming and style conventions in code-standards.md §7.

### Dependency Hygiene

**Go version:** 1.26.1 (latest stable) — matches code-standards.md §Go version requirement.

**Direct dependencies (4):**

| Package | Version | Purpose |
|---------|---------|---------|
| `github.com/BurntSushi/toml` | v1.6.0 | TOML config parsing |
| `github.com/docker/docker` | v28.5.2+incompatible | Docker SDK for container management |
| `github.com/spf13/cobra` | v1.10.2 | CLI framework |
| `github.com/stretchr/testify` | v1.11.1 | Test assertions |

All four are well-established, widely-used libraries. No risky or unvetted dependencies.

**Tool pins (2):**

| Tool | Purpose |
|------|---------|
| `github.com/daixiang0/gci` | Import sorting |
| `github.com/golangci/golangci-lint/v2/cmd/golangci-lint` | Linter runner |

Tools are managed via `go.mod` `tool` directive, updated with `go get -tool <package>@latest`. This satisfies quality-gates.md §Prerequisites: "Only Go is required."

**Indirect dependencies:** ~200+ (primarily transitive from golangci-lint's 80+ bundled linters and Docker SDK). This is expected and unavoidable for these dependencies.

**`go.sum`:** 101 KB, consistent with the transitive dependency count. No anomalies.

### CI/CD

**Status: NOT IMPLEMENTED**

No CI/CD configuration exists. Checked:
- `.github/workflows/` — absent
- `.gitlab-ci.yml` — absent
- `.circleci/` — absent
- No other CI configuration files found

**Spec requirement (quality-gates.md §CI):**
> "All gates run on every push. A failure on any gate blocks merge."

This requirement is **unmet**. Quality gates currently run only locally via:
1. Pre-commit hooks (lefthook) — developer machine only
2. `make check` — manual invocation only

**Impact:** Lint, test, and build failures can reach the main branch if a developer bypasses hooks (e.g., `--no-verify`) or pushes from a machine without lefthook installed.

### Dockerfile

**Status: NOT IMPLEMENTED**

No Dockerfile or Docker Compose files exist anywhere in the repository.

**Spec reference (havn-overview.md §Base Image):** Describes a minimal Ubuntu 24.04 image with Nix, noting "_Dockerfile and build details live in an implementation spec._" This implementation spec does not yet exist.

**Impact:** The `havn build` CLI command (defined in cli-framework.md) cannot function without a Dockerfile. This is expected for the current project stage — domain logic and CLI framework are being built before container image construction.

### Release Tooling

**Status: NOT IMPLEMENTED**

No release automation:
- No `.goreleaser.yml`
- No `CHANGELOG.md`
- No GitHub Releases configuration
- No version management (`git describe`, semver tags, etc.)

**Spec reference (havn-overview.md §Distribution):**
> "_Distribution: Nix flake in this repository (anyone can point to it). GitHub releases for Go binaries may be added later._"

**Impact:** Low for current stage. Becomes important before first public release.

### Gap Summary

| Gap | Severity | Spec Reference | Impact |
|-----|----------|---------------|--------|
| No CI/CD pipeline | **High** | quality-gates.md §CI | Quality gates not enforced on push; failures can reach main |
| No Dockerfile | **High** | havn-overview.md §Base Image | `havn build` command cannot function |
| No release tooling | **Medium** | havn-overview.md §Distribution | No automated binary distribution |
| No CHANGELOG | **Low** | — | No release history tracking |

### What Each Gap Means

**CI/CD (high):** The local tooling (Makefile, lefthook, golangci-lint) is complete and well-configured. The gap is enforcement — these gates run only on developer machines. A GitHub Actions workflow running `make check` would close this gap with minimal effort. Integration tests should be gated on Docker availability as noted in quality-gates.md.

**Dockerfile (high):** This is a feature dependency, not a tooling gap. The Dockerfile cannot be written until the base image specification is finalized. The project is correctly sequencing this — domain code first, containerization second.

**Release tooling (medium):** Not blocking current development. Should be addressed before the first public release. goreleaser + GitHub Actions is the conventional Go approach.

### Compliance Summary

| quality-gates.md Requirement | Status |
|------------------------------|--------|
| Only Go required as prerequisite | ✅ Met |
| All 8 Makefile targets present | ✅ Met |
| Targets match spec definitions | ✅ Met |
| Git hooks via lefthook | ✅ Met |
| Beads hook chaining | ✅ Met |
| `.golangci.yml` linter set matches code-standards.md | ✅ Met |
| Tool versions pinned in `go.mod` | ✅ Met |
| CI runs all gates on every push | ❌ Missing |
| CI failure blocks merge | ❌ Missing |
| Integration tests gated on Docker in CI | ❌ Missing (CI absent) |

---

## Test Coverage and Quality

_Audited: 2026-04-11 | Issue: havn-qf6.3_

Assessed against specs/test-standards.md §1–§7 and specs/quality-gates.md §2.

### Per-Package Coverage

| Package | Coverage | Test Files | Notes |
|---------|----------|------------|-------|
| `name` | 100.0% | 2 | Pure functions, fully tested |
| `volume` | 100.0% | 1 | Full coverage via fakes |
| `cli` | 91.7% | 9 | Strong — output, errors, logger, commands |
| `mount` | 91.5% | 1 | Resolve logic well-covered |
| `container` | 89.3% | 4 | Good domain coverage via fakes |
| `config` | 85.6% | 6 | Validate, resolve, flake, errors tested |
| `dolt` | 81.9% | 8 | Manager, migrate, detect, setup, config, errors |
| `doctor` | 81.0% | 4 | Runner, checks, formatting covered |
| `docker` | 55.8% | 10 | Error paths tested; success paths need integration tests |
| `cmd/havn` | 0.0% | 0 | Wiring-only entry point — expected |
| **Total** | **78.5%** | **45** | |

### Test Pattern Compliance (test-standards.md)

| Criterion | Status | Evidence |
|-----------|--------|----------|
| Black-box testing (§1) | ✅ MET | All 47 test files use `_test` package suffix |
| White-box exception documented (§1) | ✅ MET | `docker/image_stream_test.go` uses `package docker` with comment explaining why: stream-parsing edge cases best verified directly |
| Table-driven tests (§4) | ✅ MET | Used across `config/`, `name/`, `container/`, `mount/`, `docker/` |
| Testify assert/require (§4) | ✅ MET | 100% of test files use `testify`; no raw `t.Error`/`t.Fatal` |
| `require` for preconditions (§4) | ✅ MET | `require.NoError` for setup; `assert.*` for verification |
| `t.Helper()` in helpers (§4) | ✅ MET | Used in `dolt/migrate_test.go`, `docker/image_stream_test.go` |
| `t.Cleanup()` / `t.TempDir()` (§4) | ✅ MET | `t.TempDir()` throughout; explicit `t.Cleanup()` in `docker/exec_test.go` |
| Test naming `Test<Unit>_<Scenario>` (§6) | ✅ MET | All functions follow convention, e.g. `TestStart_CreatesNewContainer` |
| Subtest names lowercase phrases (§6) | ✅ MET | e.g. `"standard path"`, `"special characters sanitized"` |
| Error contracts tested (§5) | ✅ MET | `ErrorAs` checks for domain errors in `container/`, `dolt/`, `docker/` |

### Test Doubles Compliance (test-standards.md §3)

| Criterion | Status | Evidence |
|-----------|--------|----------|
| Fakes implement havn interfaces | ✅ MET | `fakeBackend`, `fakeRuntime`, `fakeStopBackend` all implement consumer-defined interfaces |
| No mocking of external APIs | ✅ MET | No test doubles implement Docker SDK interfaces |
| Fakes preferred over mocks | ✅ MET | All doubles are hand-written fakes with configurable errors and call tracking |
| No `internal/testutil/` (shared doubles) | ⚠️ N/A | Fakes defined in test files where used — appropriate for current scale |

**Fake types found:**

| Package | Fake | Implements |
|---------|------|------------|
| `container/` | `fakeImageBackend`, `fakeStopBackend`, `fakeStartBackend`, `fakeListBackend` | Consumer-defined `container.*Backend` interfaces |
| `dolt/` | `fakeBackend` (in `fake_backend_test.go`) | `dolt.Backend` |
| `volume/` | `fakeBackend` | `volume.Backend` |
| `doctor/` | `fakeCheck`, `blockingFakeCheck` | `doctor.Check` |
| `cli/` | Various fakes per command test file | Command-specific interfaces |

### Integration Test Infrastructure

**Status: DEFINED BUT EMPTY**

- `Makefile` has `test-integration` target: `go test -tags integration ./...`
- `specs/test-standards.md` documents the `//go:build integration` pattern
- **No test files carry the `integration` build tag** — zero integration tests exist

This means the wrapper layer (`internal/docker/`) has no tests against a real Docker daemon. The gap is mitigated by:
1. Domain packages are well-tested via fakes (89–100% coverage)
2. Docker wrapper error paths are tested via unreachable daemon
3. Error types and boundary translation are fully tested

However, success-path behavior (correct Docker API translation, response mapping, filter behavior) is unverified. This is the primary coverage gap.

### Docker Package Deep Dive (55.8%)

The docker package is the infrastructure wrapper — it translates between havn domain types and the Docker SDK. Its 55.8% coverage is the lowest non-trivial package.

**What IS tested (unit-testable without Docker):**

| Area | Coverage | Approach |
|------|----------|----------|
| Error types (6 types, 18 methods) | 100% | Direct construction and assertion |
| `EnvSlice`, `BuildMounts` helpers | 100% | Pure function tests |
| `ParseMemoryBytes` | 92.3% | Table-driven with 7 cases |
| `TerminalFd` | 100% | File descriptor detection |
| `streamBuildOutput` (internal) | 85.7% | White-box test for JSON stream parsing |
| `tarDir`, `copyFileToTar` (internal) | 74–86% | White-box test for tar creation |

**What is tested — error paths only (no success cases):**

| Function | Coverage | Gap |
|----------|----------|-----|
| `ContainerCreate` | 66.7% | Success path: response → ID mapping untested |
| `ContainerStart` | 66.7% | Success path untested |
| `ContainerStop` | 71.4% | Success path untested |
| `ContainerRemove` | 66.7% | Success path untested |
| `ContainerList` | 25.9% | Filter translation, response mapping untested |
| `ContainerInspect` | 18.2% | State/port/mount mapping untested |
| `CopyToContainer` | 66.7% | Success path untested |
| `CopyFromContainer` | 66.7% | Success path untested |
| `NetworkCreate` | 66.7% | Success path, `ErrNetworkAlreadyExists` idempotency untested |
| `NetworkInspect` | 44.4% | Subnet/gateway mapping untested |
| `NetworkList` | 40.0% | Filter behavior untested |
| `VolumeInspect` | 66.7% | Mountpoint/label mapping untested |
| `VolumeCreate` | 75.0% | Label propagation untested |
| `VolumeList` | 43.8% | Filter behavior untested |
| `ImageInspect` | 44.4% | Metadata mapping untested |
| `ImageExists` | 66.7% | True/false return logic untested |
| `ImageBuild` | 70.4% | Full build flow untested |

**Completely untested (0%):**

| Function | Reason |
|----------|--------|
| `handleSIGWINCH` | Signal handling — requires terminal and running container |
| `resizeExec` | Called by `handleSIGWINCH` — same constraint |

**`ContainerAttach`** has 14.8% coverage — only the initial error path is tested. The interactive I/O flow (stdin/stdout proxying, raw terminal mode, signal forwarding) is untested. This function is inherently difficult to unit test and is a strong candidate for integration/system tests.

### Why 55.8% is Expected

The docker package is a **wrapper** (code-standards.md §4). Its primary job is type translation and Docker API calls. Testing success paths requires a running Docker daemon, which makes them **integration tests** by definition (test-standards.md §2). The current unit tests correctly cover what can be verified without Docker:

1. Error handling and boundary translation
2. Pure helper functions
3. Context cancellation propagation
4. Error type implementation

The missing success-path tests belong in `//go:build integration` tagged files, which don't exist yet.

### Identified Gaps

| Gap | Severity | Spec Reference |
|-----|----------|---------------|
| No integration tests for docker wrapper success paths | **High** | test-standards.md §2: "Verify that boundaries work in practice" |
| No `testdata/` directories | **Low** | test-standards.md §1: Convention documented but no test data files needed yet |
| No shared test doubles in `internal/testutil/` | **Low** | test-standards.md §3: Current scale doesn't require shared doubles |
| `cmd/havn` has 0% coverage | **Low** | Wiring-only entry point; tested indirectly through `cli` package |
| `handleSIGWINCH` / `resizeExec` untested | **Medium** | Terminal signal handling — needs integration test with PTY |
| `ContainerAttach` mostly untested (14.8%) | **Medium** | Interactive I/O flow — strong integration test candidate |

### Compliance Summary

| Area | Compliance | Notes |
|------|-----------|-------|
| Test organization (§1) | ✅ Full | Files next to code, `_test` suffix, documented exception |
| Test boundaries (§2) | ⚠️ Partial | Unit tests excellent; integration tests absent |
| Test doubles (§3) | ✅ Full | Fakes implement havn interfaces, not external APIs |
| Test patterns (§4) | ✅ Full | Table-driven, testify, helpers, cleanup all correct |
| Contract testing (§5) | ✅ Full | Error contracts verified with `ErrorAs` |
| Naming (§6) | ✅ Full | All functions and subtests follow conventions |
| CI integration (§7) | ❌ Missing | No CI pipeline exists (see Infrastructure section) |

### Recommendations (audit only — no code changes)

1. **Create integration tests** for `internal/docker/` success paths behind `//go:build integration`. Priority functions: `ContainerList`, `ContainerInspect`, `NetworkInspect`, `VolumeList` (lowest coverage, most complex translation logic).
2. **Add `ContainerAttach` integration test** with PTY simulation to verify interactive session flow.
3. **Consider `internal/testutil/`** if fakes begin duplicating across packages as the codebase grows.
4. **`cmd/havn` coverage** is not a concern — the entry point delegates immediately to `cli.Execute()` which is well-tested.
