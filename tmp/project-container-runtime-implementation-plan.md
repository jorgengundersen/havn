# Implementation plan: project container runtime path split

Date: 2026-05-08

Tracking issue: `havn-2wb` — Implement project container runtime path split

This document is an implementation plan, not the task tracker. The durable task
record remains the beads issue above; newly discovered work should be filed in
`bd` rather than added here as an open-ended TODO list.

## Goal

Implement the contract in `specs/project-container-runtime.md` so `havn`
distinguishes host project paths from in-container project paths:

```text
HostProjectPath      = absolute project path on the host
ContainerProjectPath = /home/devuser/<path relative to host home>
```

The implementation must preserve host-path semantics for host-side identity and
discovery, while using the container project path for all in-container working
directories, mount targets, runtime probes, and user-facing in-container
instructions.

## Governing specs and standards

- [ ] `specs/project-container-runtime.md`
  - authoritative host/container project path mapping
  - project bind-mount source/target contract
  - labels and old-layout drift detection
- [ ] `specs/configuration.md`
  - host project path boundary under host home
  - host-side config discovery and effective config semantics
- [ ] `specs/cli-framework.md`
  - CLI path field semantics
  - startup/entry behavior
  - JSON/error framing
- [ ] `specs/environment-interface.md`
  - in-container execution context and command working directory
- [ ] `specs/havn-doctor.md`
  - doctor host/container path split
  - check catalog and output semantics
- [ ] `specs/base-image.md`
  - fixed `devuser` account and `/home/devuser` runtime home
- [ ] `specs/code-standards.md`
  - pure path/mount helpers
  - domain-first packages
  - no ambiguous `projectPath string` where both path meanings are needed
  - dependency isolation from Docker SDK types
- [ ] `specs/test-standards.md`
  - red/green TDD
  - black-box tests by default
  - distinct host/container home fixtures

## Non-negotiable TDD workflow

Follow red/green/refactor as a strict micro-cycle. Do not batch tests.

For each behavior below:

- [ ] Red: write exactly one failing test for the next behavior
- [ ] Green: implement the smallest change that makes that test pass
- [ ] Refactor: clean up while all tests stay green
- [ ] Commit or continue only while the focused test set stays green
- [ ] Repeat for the next behavior

A phase may list multiple behaviors, but those behaviors are not a batch. Each
checkbox represents its own TDD cycle.

Use fixtures that make path confusion visible:

```text
host home:      /home/alice
host project:   /home/alice/work/api
container home: /home/devuser
container path: /home/devuser/work/api
```

Do not use `/home/devuser` as a fake host home unless that specific edge case is
being tested.

## Phase 0 — Preparation

- [ ] Claim the issue:

  ```bash
  bd update havn-2wb --claim --json
  ```

- [ ] Review the relevant specs before implementation:
  - [ ] `specs/project-container-runtime.md`
  - [ ] `specs/configuration.md`
  - [ ] `specs/cli-framework.md`
  - [ ] `specs/environment-interface.md`
  - [ ] `specs/havn-doctor.md`
  - [ ] `specs/code-standards.md`
  - [ ] `specs/test-standards.md`

- [ ] Review likely implementation files before editing:
  - [ ] `internal/cli/project_context.go`
  - [ ] `internal/mount/resolve.go`
  - [ ] `internal/container/start.go`
  - [ ] `internal/container/enter.go`
  - [ ] `internal/container/list.go`
  - [ ] `internal/docker/container.go`
  - [ ] `internal/cli/adapters_start.go`
  - [ ] `internal/cli/doctor.go`
  - [ ] `internal/doctor/container_checks.go`

## Phase 1 — Add explicit project path model

Objective: introduce a pure path model that carries both host and container
project paths.

Preferred shape, adjusted to existing package conventions as needed:

```go
type Paths struct {
    HostPath      string
    ContainerPath string
}

func ContainerProjectPath(hostHome, hostProjectPath string) (string, error)
func Resolve(hostHome, hostProjectPath string) (Paths, error)
```

Potential package: `internal/projectpath`, unless an existing domain package is
clearly more appropriate.

TDD cycles:

- [ ] Red: test `/home/alice/work/api` maps to `/home/devuser/work/api`.
- [ ] Green: implement minimal `ContainerProjectPath` mapping.
- [ ] Refactor.

- [ ] Red: test project paths outside host home are rejected.
- [ ] Green: add escape validation for `..` and `../...` relative paths.
- [ ] Refactor.

- [ ] Red: test host project equal to host home maps to `/home/devuser`.
- [ ] Green: handle `.` relative path correctly.
- [ ] Refactor.

- [ ] Red: test `Resolve` returns both unchanged `HostPath` and derived
  `ContainerPath`.
- [ ] Green: add `Paths`/`Resolve` helper.
- [ ] Refactor.

Completion criteria:

- [ ] Path helper is pure and unit-tested.
- [ ] No Docker or filesystem side effects are introduced.
- [ ] Error message is actionable enough for CLI wrapping.

## Phase 2 — Extend CLI project context

Objective: make CLI project context carry both paths while keeping host path as
the source of truth for config and identity.

Host path remains authoritative for:

- project config discovery
- host flake discovery
- deterministic container name derivation
- default Dolt database derivation
- stop/list/lookup identity
- `havn up --json` `project_path`

TDD cycles:

- [ ] Red: project context exposes canonical resolved host path unchanged.
- [ ] Green: add explicit host path field or `Paths.HostPath` while preserving
  current behavior.
- [ ] Refactor.

- [ ] Red: project context computes `/home/devuser/...` container path from host
  home and host project path with host home controlled by the test.
- [ ] Green: wire project path helper into project context resolution and keep host
  home lookup injectable/testable rather than hidden in low-level helpers.
- [ ] Refactor.

- [ ] Red: project config discovery still uses host project path.
- [ ] Green: update config call sites to use `HostPath` explicitly.
- [ ] Refactor.

- [ ] Red: container name derivation still uses host project path.
- [ ] Green: update naming call sites to use `HostPath` explicitly.
- [ ] Refactor.

Completion criteria:

- [ ] No CLI orchestration code passes ambiguous path values into runtime layers
  that need both meanings.
- [ ] Existing host-home boundary behavior remains intact.

## Phase 3 — Update mount resolution

Objective: change the project bind mount from host-path target to container-path
target.

Desired project mount:

```text
Source: <HostProjectPath>
Target: <ContainerProjectPath>
Type:   bind
```

TDD cycles:

- [ ] Red: project directory mount source is `/home/alice/work/api` and target
  is `/home/devuser/work/api`.
- [ ] Green: update mount resolver API/logic to accept explicit project paths.
- [ ] Refactor.

- [ ] Red: config mounts still map host-home-relative paths into
  `/home/devuser/...`.
- [ ] Green: preserve existing config mount target behavior after API changes.
- [ ] Refactor.

- [ ] Red: named volume targets remain unchanged:
  - `/nix`
  - `/home/devuser/.local/share`
  - `/home/devuser/.cache`
  - `/home/devuser/.local/state`
- [ ] Green: fix any regression from mount resolver changes.
- [ ] Refactor.

Completion criteria:

- [ ] Project bind mount uses host source and container target.
- [ ] Existing config mounts and named volumes remain compatible.

## Phase 4 — Update startup and interactive attach

Objective: use host path for identity and container path for interactive shell
workdir.

Likely files:

- `internal/container/start.go`
- `internal/container/start_test.go`
- `internal/cli/adapters_start.go`

TDD cycles:

- [ ] Red: creating a new project container receives a project mount with target
  `/home/devuser/work/api`.
- [ ] Green: pass explicit project paths from CLI/start orchestration into mount
  resolution and create opts.
- [ ] Refactor.

- [ ] Red: interactive attach for `havn [path]` uses workdir
  `/home/devuser/work/api`.
- [ ] Green: pass `ContainerPath` to interactive exec.
- [ ] Refactor.

- [ ] Red: project container name still derives from `/home/alice/work/api`.
- [ ] Green: keep name derivation tied to `HostPath`.
- [ ] Refactor.

- [ ] Red: `havn up --json` still reports host `project_path`.
- [ ] Green: keep CLI output field host-path based.
- [ ] Refactor.

Completion criteria:

- [ ] `havn [path]` attaches from container project path.
- [ ] `havn up [path]` creates containers with new mount layout.
- [ ] Host path identity is preserved.

## Phase 5 — Add non-interactive exec workdir support

Objective: make startup validation and preparation run from the container project
path.

Docker already supports exec workdir; the domain interfaces should expose that
without leaking Docker SDK types.

Preferred shape:

```go
type ExecOpts struct {
    Cmd     []string
    Workdir string
}
```

TDD cycles:

- [ ] Red: startup validation exec receives workdir `/home/devuser/work/api`.
- [ ] Green: add workdir-aware non-interactive exec interface and adapter
  plumbing.
- [ ] Refactor.

- [ ] Red: startup prepare exec receives workdir `/home/devuser/work/api`.
- [ ] Green: wire prepare phase through workdir-aware exec.
- [ ] Refactor.

- [ ] Red: missing optional prepare capability remains non-fatal.
- [ ] Green: preserve existing optional-capability behavior while using workdir.
- [ ] Refactor.

- [ ] Red: executed prepare failure remains fatal and actionable.
- [ ] Green: preserve existing failure semantics.
- [ ] Refactor.

Completion criteria:

- [ ] Validation and prepare commands run from `ContainerProjectPath`.
- [ ] No unsafe shell `cd` string concatenation is introduced when Docker workdir
  can be used.

## Phase 6 — Update `havn enter`

Objective: use host path for lookup/name and container path for plain bash
working directory.

TDD cycles:

- [ ] Red: `havn enter [path]` attaches plain bash with workdir
  `/home/devuser/work/api`.
- [ ] Green: pass `ContainerPath` to enter interactive exec.
- [ ] Refactor.

- [ ] Red: missing container guidance references host path in `havn up <path>`.
- [ ] Green: preserve host-path user guidance.
- [ ] Refactor.

- [ ] Red: stopped container guidance references host path in `havn up <path>`.
- [ ] Green: preserve host-path user guidance.
- [ ] Refactor.

Completion criteria:

- [ ] `havn enter` starts `bash` from `ContainerProjectPath`.
- [ ] User-facing lifecycle guidance remains host-path based.

## Phase 7 — Add container project path label

Objective: preserve `havn.path` as host path and add `havn.path.container`.

Required labels:

```text
havn.path           = <HostProjectPath>
havn.path.container = <ContainerProjectPath>
```

TDD cycles:

- [ ] Red: newly created containers still include `havn.path` as host path.
- [ ] Green: ensure creation labels use `HostPath` explicitly.
- [ ] Refactor.

- [ ] Red: newly created containers include `havn.path.container` as container
  path.
- [ ] Green: add `LabelContainerPath` constant and creation label.
- [ ] Refactor.

- [ ] Red: `havn list` human output still reports host project path.
- [ ] Green: keep list display tied to `havn.path`.
- [ ] Refactor.

- [ ] Red: list handling does not reinterpret old `havn.path` as container path
  when `havn.path.container` is absent.
- [ ] Green: make container path optional in list state/output.
- [ ] Refactor.

Completion criteria:

- [ ] `havn.path` remains host path.
- [ ] New containers publish `havn.path.container`.
- [ ] List behavior remains backward-compatible.

## Phase 8 — Extend inspect state for labels and mounts

Objective: expose enough inspect data to detect layout drift while preserving
Docker dependency isolation.

Target domain shape, adjusted to existing naming:

```go
type State struct {
    ID      string
    Running bool
    Labels  map[string]string
    Mounts  []Mount
}

type Mount struct {
    Source string
    Target string
    Type   string
}
```

TDD cycles:

- [ ] Red: container inspect adapter exposes labels to domain state.
- [ ] Green: extend domain `State` and map Docker labels at adapter boundary.
- [ ] Refactor.

- [ ] Red: container inspect adapter exposes mounts to domain state.
- [ ] Green: map Docker mounts into domain mount structs.
- [ ] Refactor.

- [ ] Red: domain package has no Docker SDK type imports.
- [ ] Green: keep all Docker translation in wrapper/adapter code.
- [ ] Refactor.

Completion criteria:

- [ ] Startup/enter logic can inspect labels and mounts without Docker SDK
  imports.
- [ ] Existing tests using minimal state remain easy to adapt.

## Phase 9 — Implement project mount layout drift detection

Objective: reject old-layout containers before start or attach.

Detection rule:

- find an existing project container for the host project;
- inspect its project bind mount by host source and/or labels;
- expected target is `ContainerProjectPath`;
- if actual target differs, fail with actionable drift error.

Suggested typed error:

```go
type ProjectMountLayoutDriftError struct {
    ContainerName   string
    HostProjectPath string
    ExpectedTarget  string
    ActualTarget    string
}
```

Suggested stable JSON error type:

```text
project_mount_layout_drift
```

TDD cycles:

- [ ] Red: running old-layout container fails before interactive attach.
- [ ] Green: validate mount layout before attach.
- [ ] Refactor.

- [ ] Red: stopped old-layout container fails before start.
- [ ] Green: validate mount layout before starting stopped containers.
- [ ] Refactor.

- [ ] Red: correct-layout running container attaches successfully.
- [ ] Green: allow matching mount target.
- [ ] Refactor.

- [ ] Red: correct-layout stopped container starts successfully.
- [ ] Green: allow matching mount target before start.
- [ ] Refactor.

- [ ] Red: missing project mount returns actionable drift/layout error.
- [ ] Green: handle missing mount as incompatible layout.
- [ ] Refactor.

- [ ] Red: JSON error details include container, host project path, expected
  target, and actual target when known.
- [ ] Green: implement typed error details.
- [ ] Refactor.

Completion criteria:

- [ ] Old-layout containers are not silently reused.
- [ ] Error guidance tells users to recreate through supported havn lifecycle
  paths.
- [ ] Raw `docker rm -f` is not the primary recommendation unless no havn-native
  path exists.

## Phase 10 — Update doctor path handling

Objective: doctor uses host paths for host checks and container paths for
container checks.

Target doctor model:

```go
type doctorContainerTarget struct {
    HostProjectPath      string
    ContainerProjectPath string
    // other existing fields
}
```

or use the shared project path model.

TDD cycles:

- [ ] Red: `project_mount` probes `/home/devuser/work/api`.
- [ ] Green: pass container project path into project mount check.
- [ ] Refactor.

- [ ] Red: host `.beads/` existence decision uses
  `/home/alice/work/api/.beads`.
- [ ] Green: keep host-side beads detection on `HostProjectPath`.
- [ ] Refactor.

- [ ] Red: `doctor --all` treats `havn.path` as host project path.
- [ ] Green: update target modeling and label interpretation.
- [ ] Refactor.

- [ ] Red: `doctor --all` uses `havn.path.container` when present.
- [ ] Green: read optional container path label.
- [ ] Refactor.

- [ ] Red: when `havn.path.container` is missing and `havn.path` is under the
  current host home, doctor computes expected container path from `havn.path`
  and host home.
- [ ] Green: add fallback computation.
- [ ] Refactor.

- [ ] Red: when `havn.path.container` is missing and `havn.path` is not under the
  current host home, doctor reports an actionable diagnostic instead of
  miscomputing or failing the entire report.
- [ ] Green: add graceful fallback/error reporting for non-computable container
  paths.
- [ ] Refactor.

- [ ] Red: old-layout project mount is reported diagnostically with path detail
  fields.
- [ ] Green: inspect mounts and add doctor result detail/recommendation.
- [ ] Refactor.

- [ ] Red: `beads_health` runs from `/home/devuser/work/api`.
- [ ] Green: add or reuse workdir-aware doctor exec.
- [ ] Refactor.

Completion criteria:

- [ ] Host-tier checks use host project path.
- [ ] Container-tier checks use container project path.
- [ ] Beads health runs with `PWD=ContainerProjectPath` and `HOME=/home/devuser`.
- [ ] Doctor reports old-layout drift instead of treating it as healthy.

## Phase 11 — Decide and implement Docker create working directory if needed

Objective: decide whether container default `WorkingDir` should be set to
`ContainerProjectPath` in addition to explicit exec workdirs.

The spec requires havn-managed execs to use `ContainerProjectPath`; Docker
container default `WorkingDir` is not strictly required. It may still be useful
for coherent defaults.

TDD cycles, if implemented:

- [ ] Red: container create opts include workdir `/home/devuser/work/api`.
- [ ] Green: add `Workdir` to create opts and set it during startup.
- [ ] Refactor.

- [ ] Red: Docker adapter passes create workdir into Docker config.
- [ ] Green: wire adapter field to Docker `WorkingDir`.
- [ ] Refactor.

Completion criteria:

- [ ] Decision is documented in code or commit message.
- [ ] Havn-managed exec correctness does not depend solely on Docker default
  workdir.

## Phase 12 — Update or add boundary/contract tests

Objective: ensure the new runtime contract is covered at appropriate levels.

Execute as micro-cycles. Do not add all tests at once.

Candidate coverage:

- [ ] Red/green/refactor: mount layout contract.
- [ ] Red/green/refactor: attach workdir contract.
- [ ] Red/green/refactor: enter workdir contract.
- [ ] Red/green/refactor: startup validation workdir contract.
- [ ] Red/green/refactor: startup prepare workdir contract.
- [ ] Red/green/refactor: label contract.
- [ ] Red/green/refactor: inspect labels/mounts contract.
- [ ] Red/green/refactor: drift detection contract.
- [ ] Red/green/refactor: doctor split-path contract.
- [ ] Red/green/refactor: beads-health CWD contract.

Completion criteria:

- [ ] Tests avoid fake host paths under `/home/devuser` unless intentionally
  testing that edge case.
- [ ] Tests assert behavior and stable outputs, not prose from specs/docs.

## Phase 13 — Update specs and derivative docs after behavior lands

Only update support-status labels and derivative user docs after implementation
is real. Do not present partial behavior as shipped.

Spec updates to consider after the runtime behavior passes quality gates:

- [ ] `specs/project-container-runtime.md`: move from `Partial` to `Implemented`
  only when the full contract is shipped.
- [ ] `specs/cli-framework.md`: update command support rows for startup/enter and
  doctor only when those surfaces satisfy their path-layout contracts.
- [ ] `specs/havn-doctor.md`: move back to `Implemented` only when doctor path
  behavior satisfies the updated contract.

Likely files:

- [ ] `README.md`
- [ ] `docs/cli-reference.md`
- [ ] `docs/doctor-troubleshooting.md`
- [ ] any Dolt/beads guide mentioning current path workarounds

Doc updates should explain:

- [ ] host path vs container path
- [ ] projects mount under `/home/devuser/<host-home-relative-path>`
- [ ] `havn list` shows host paths by default
- [ ] old-layout containers need recreation
- [ ] beads should no longer require `HOME=/home/<host-user>` workarounds inside
  the project container

## Phase 14 — Quality gates

Run targeted tests throughout the work. Examples:

```bash
go test ./internal/projectpath ./internal/mount
go test ./internal/container
go test ./internal/cli
go test ./internal/doctor
```

Before final completion:

- [ ] Run the main gate:

  ```bash
  make check
  ```

- [ ] If Docker-backed behavior changed and the environment supports it, run:

  ```bash
  make test-integration
  make test-boundary-confidence
  ```

If Docker, network, registry, or mirror failures occur, report them as
environment failures. Do not bake host-specific workarounds into shared project
configuration.

## Suggested conventional commit sequence

Keep commits small and green. Suggested sequence:

- [ ] `feat: add project path runtime model`
- [ ] `feat: mount projects under container user home`
- [ ] `feat: use container project path for startup execs`
- [ ] `feat: use container project path for enter`
- [ ] `feat: label project containers with container path`
- [ ] `feat: detect project mount layout drift`
- [ ] `feat: split doctor host and container project paths`
- [ ] `test: cover project container path layout`
- [ ] `docs: update project path runtime docs`

Smaller commits are fine if each commit represents a coherent green state.

## Completion criteria

Do not close `havn-2wb` until all relevant items are satisfied:

- [ ] Host and container project paths are distinct in domain models.
- [ ] Project bind source is host project path.
- [ ] Project bind target is `/home/devuser/<host-home-relative-path>`.
- [ ] `havn [path]` interactive workdir is container project path.
- [ ] `havn enter [path]` workdir is container project path.
- [ ] Startup validation and prepare commands run from container project path.
- [ ] New containers have `havn.path=<HostProjectPath>`.
- [ ] New containers have `havn.path.container=<ContainerProjectPath>`.
- [ ] Existing old-layout containers are detected and rejected before attach or
  start.
- [ ] Drift errors are actionable and structured for JSON mode where applicable.
- [ ] Doctor uses host path for host checks and container path for container
  checks.
- [ ] Beads health runs from the container project path.
- [ ] `havn up --json` keeps `project_path` as host project path.
- [ ] Tests use distinct host/container home fixtures.
- [ ] `make check` passes.
- [ ] Integration/boundary-confidence tests are run or skipped with clear
  environment rationale.
- [ ] Derivative docs are updated only once behavior is actually implemented.
- [ ] Changes are committed with conventional commits.
- [ ] Issue `havn-2wb` is closed with a completion reason after verification.
- [ ] Work is pushed to remote and `git status` shows up to date with origin.
