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
