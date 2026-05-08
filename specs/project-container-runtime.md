# Project Container Runtime

This is the authoritative runtime contract for project container filesystem
layout, project path mapping, container labels, and existing-container layout
compatibility.

Status: Partial

`Partial` means this document is the intended contract, but the current
implementation may not yet satisfy every detail. Derivative docs must call out
current gaps instead of presenting the whole contract as shipped.

## Ownership

This spec owns:

- the split between host project paths and in-container project paths
- the project bind-mount source and target contract
- the working-directory contract for havn-managed in-container commands
- project-container path labels used for lookup, diagnostics, and drift checks
- old-layout project mount drift detection and remediation guidance

This spec does not own:

- host project-path discovery, precedence, and validation before startup; those
  are owned by `specs/configuration.md`
- CLI command tree, flag scope, stream separation, and JSON framing; those are
  owned by `specs/cli-framework.md`
- environment flake entrypoints and optional startup preparation capability;
  those are owned by `specs/environment-interface.md`
- the base-image `devuser` account, UID/GID mapping, and image filesystem
  prerequisites; those are owned by `specs/base-image.md`
- doctor check catalog and doctor exit semantics; those are owned by
  `specs/havn-doctor.md`

## Path Model

Project runtime code must distinguish two paths:

- **Host project path**: the resolved absolute project path on the host running
  `havn`.
- **Container project path**: the absolute path where that project is mounted
  inside the project container.

These paths are not necessarily equal.

A runtime path value that may be used for both host and container concerns must
be modeled explicitly, for example:

```go
type ProjectPaths struct {
    HostPath      string
    ContainerPath string
}
```

Passing a bare `projectPath string` through layers that need both meanings is
not part of this contract.

## Path Mapping Rule

Given:

```text
hostHome = the host user's home directory
hostProjectPath = resolved absolute project path on the host
containerHome = /home/devuser
```

`havn` computes:

```text
relativeProjectPath = filepath.Rel(hostHome, hostProjectPath)
containerProjectPath = filepath.Join(containerHome, relativeProjectPath)
```

Example:

```text
hostHome:             /home/alice
hostProjectPath:      /home/alice/work/api
relativeProjectPath:  work/api
containerHome:        /home/devuser
containerProjectPath: /home/devuser/work/api
```

The relative project path must not escape `hostHome`. If the relative path is
`..` or begins with `../`, startup fails before project container creation.

`specs/configuration.md` owns the startup rule that host project paths for
`havn [path]` and `havn up [path]` must resolve under the host user's home
directory. This spec owns what runtime does with that valid host path.

## Host Project Path Uses

`havn` uses the host project path for host-side identity and discovery:

- Docker bind-mount source
- project config discovery, including `<project>/.havn/config.toml`
- project flake discovery on the host
- deterministic project container name derivation
- default shared-Dolt database basename
- host-side lifecycle lookup and stop/list identity
- the `havn.path` Docker label

`havn list` reports host project paths by default. If a command exposes the
in-container project path, it must use a separate field or label rather than
reinterpreting the host path.

## Container Project Path Uses

`havn` uses the container project path for in-container runtime behavior:

- Docker bind-mount target
- interactive exec working directory for `havn [path]`
- interactive exec working directory for `havn enter [path]`
- non-interactive startup validation and preparation command working directory
- in-container doctor project-mount and beads-health checks
- user-facing in-container remediation instructions

Havn-managed in-container project commands run with:

```text
USER/HOME user: devuser
HOME:           /home/devuser
PWD:            <containerProjectPath>
```

`havn` must not set `HOME` to the host user's home directory as the primary fix
for host/container path mismatch.

## Project Bind Mount Contract

The project bind mount is:

```text
Source: <hostProjectPath>
Target: <containerProjectPath>
Type:   bind
```

For example:

```text
Source: /home/alice/work/api
Target: /home/devuser/work/api
Type:   bind
```

The old behavior of using the host project path as both source and target is not
part of the supported runtime layout.

Other configured host-home-relative config mounts continue to map into the
runtime user's home under `/home/devuser`. Named volumes continue to mount at
their configured runtime locations, including:

```text
/nix
/home/devuser/.local/share
/home/devuser/.cache
/home/devuser/.local/state
```

## Container Labels

Project containers use labels to preserve host-side identity and publish the
runtime layout:

```text
havn.path           = <hostProjectPath>
havn.path.container = <containerProjectPath>
```

`havn.path` remains the host project path for backward-compatible lookup,
listing, and project identity. It must not be reinterpreted as an in-container
path.

`havn.path.container` records the expected in-container project path for drift
detection and diagnostics. Older containers may not have this label; in that
case `havn` computes the expected container path from `havn.path` and the host
home when possible, and may inspect mount metadata to report the actual target.

## Existing Container Layout Compatibility

Docker bind-mount targets are create-time container configuration. `havn` must
not silently reuse an existing project container whose project mount target does
not match the expected container project path.

When an existing project container is found, startup and entry workflows must:

1. inspect labels and mount metadata when available;
2. compute the expected container project path for the host project;
3. verify that the host project is mounted at that expected target;
4. fail before attach/start if the container uses an incompatible layout; and
5. report actionable recreation guidance.

The drift error should include, when known:

- host project path
- current or actual project mount target
- expected container project path
- the affected container name
- supported recreation guidance

Existing stopped containers with drift are not fixable by starting them.
Existing running containers with drift are not safe to attach to as if the new
layout were active.

## Command Surface Implications

`havn [path]`, `havn up [path]`, and `havn enter [path]` accept and report
host-side project paths at the CLI boundary unless a field explicitly says it is
a container path.

`havn up --json`'s `project_path` field is the resolved host project path. If a
future or current JSON payload includes the in-container path, the field must be
named distinctly, for example `container_project_path`.

Interactive entry commands execute inside the project container from the
container project path:

- `havn [path]`: `nix develop <env>#<shell>` from `<containerProjectPath>`
- `havn enter [path]`: plain `bash` from `<containerProjectPath>`

## Doctor Implications

Doctor uses split paths:

- host-tier config, project config, flake discovery, and `.beads/` existence
  decisions use the host project path;
- container-tier project mount, config mount target, dev-shell, and beads-health
  probes use the container project path.

`project_mount` verifies that the project is mounted and writable at the
expected container project path. `beads_health` runs from the container project
path so beads observes the repository under `/home/devuser` while `HOME` is
also `/home/devuser`.

## Non-Goals

This contract does not require the container username to match the host
username. The supported model is the fixed runtime user `devuser` with numeric
UID/GID matching the host user, as defined by `specs/base-image.md`.

This contract does not support mutating existing containers in place to change
project bind-mount targets. Recreate the project container through supported
havn lifecycle commands when layout drift is reported.

This contract does not weaken beads path-safety checks or configure beads to
accept repositories under another user's home directory.

## Relationship To Other Specs

- `specs/configuration.md` owns host project-path resolution and validation.
- `specs/cli-framework.md` owns command output and error framing.
- `specs/environment-interface.md` owns flake entrypoints and preparation
  capability semantics, while this spec owns their in-container working
  directory.
- `specs/base-image.md` owns the `devuser` account and `/home/devuser`
  filesystem prerequisites.
- `specs/havn-doctor.md` owns check selection, check identifiers, and doctor
  output, while this spec owns the path layout those checks verify.
