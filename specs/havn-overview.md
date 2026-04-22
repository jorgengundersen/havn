# havn Overview

This document is the product overview for `havn`.

Status: Partial

It explains product shape, primary workflows, and reading order. It is not the
authoritative owner for detailed configuration, CLI, doctor, or shared-Dolt
contracts.

## Product Shape

`havn` is a Go CLI for reproducible development environments built on Docker and
Nix. It manages one container per project, reuses shared host-side
infrastructure where possible, and supports separate startup and entry
workflows.

At a high level, `havn` owns:

- project path resolution under the user's home directory
- deterministic project-container naming
- shared infrastructure setup such as the network, volumes, and base image
- start-or-attach behavior for project containers
- optional shared-Dolt wiring for projects that use beads

`havn` does not own the contents of the development environment itself. The dev
environment comes from the selected Nix flake and shell.

## Authoritative Subsystem Specs

- Configuration discovery, precedence, merge rules, and `havn config show`:
  `specs/configuration.md`
- Command tree, flag scope, output modes, JSON contracts, and CLI error
  handling: `specs/cli-framework.md`
- Environment flake entrypoints and optional startup preparation capability:
  `specs/environment-interface.md`
- Doctor checks, tiering, selection rules, and doctor output:
  `specs/havn-doctor.md`
- Shared Dolt lifecycle, readiness, ownership, migration, and safety semantics:
  `specs/shared-dolt-server.md`
- Base image and runtime assumptions: `specs/base-image.md`

## Core Workflow

### Workflow surfaces

At overview level, the user-facing workflow split is:

- `havn [path]` (implemented): start or attach, then enter the configured Nix dev shell
- `havn up [path]` (implemented): lifecycle startup only (no interactive attach)
- `havn enter [path]` (implemented): plain interactive shell entry (`bash`) without automatic `nix develop`

Environment startup-preparation contract for this split:

- startup preparation is environment-owned via optional capability entrypoint
  defined in `specs/environment-interface.md`
- the environment-interface contract is ratified at `Status: Partial`; any
  remaining gaps are runtime-alignment follow-up
- primary interactive startup (`havn [path]`) runs preparation when available
  and is fail-closed if that prepare step runs and fails
- `havn up [path]` remains non-interactive and lifecycle-focused by default; it
  does not run optional startup preparation unless explicitly requested
- `havn up [path] --prepare` runs optional startup preparation when available
  and returns a command error on prepare failure
- `havn enter [path]` remains plain-shell entry and does not run startup
  preparation
- ad-hoc `nix develop` from inside entered sessions remains supported

`havn up [path]` uses the same startup override surface as
`havn [path]` for `--env`, `--cpus`, `--memory`, `--port`, `--no-dolt`, and
`--image`. `havn up [path]` also supports startup-check modifiers
`--validate` and `--prepare`. `--shell` remains exclusive to `havn [path]`
because `up` does not start an interactive shell session.

`havn enter [path]` requires the project container to already be running. If the
container is missing or stopped, the command returns actionable guidance to run
`havn up <path>`.

When entering an existing running container, `havn enter [path]` also performs
Nix registry persistence preparation before shell entry, so users do not need a
prior startup run just to make in-container `nix registry` alias changes
persist.

### Start or attach

The primary entry point is:

```text
havn [path]
```

Default path is `.`.

At overview level, startup works like this:

1. resolve the target project path
2. resolve effective configuration for that project
3. derive the deterministic container name
4. attach if the project container is already running
5. otherwise ensure shared prerequisites exist and start the project container
6. exec into the configured dev shell

`havn up [path]` runs the same startup orchestration through step 5 and
then exits without entering a shell. Optional startup-check flags may add
non-interactive validation/preparation phases before command completion.

On successful attach, the root command exits with the shell session's exit code.

Detailed config resolution lives in `specs/configuration.md`. Detailed CLI and
error behavior lives in `specs/cli-framework.md`.

Startup observability at overview level:

- baseline startup diagnostics are retained by default for investigation
- default terminal output remains concise
- `--verbose` is the opt-in mode for detailed startup diagnostics in terminal

### Shared infrastructure

Startup may create or reuse:

- the configured Docker network
- named Docker volumes for Nix and XDG state
- the configured base image
- the shared Dolt server when project config enables it

These resources are shared across projects. They are not torn down as part of a
single project startup failure.

When multiple projects share the same state volume (`volumes.state`), they also
share Nix registry aliases persisted from in-container `nix registry` commands.

### Stop and maintenance workflows

At overview level, `havn` provides command surfaces for:

- stopping one project container or stopping all havn-managed project containers
- listing managed containers and shared volumes
- building the base image
- inspecting effective configuration
- running diagnostics (`doctor`)
- managing shared Dolt lifecycle and data migration

Exact command names, flag scope, output semantics, support status, and
best-effort behavior are owned by `specs/cli-framework.md`,
`specs/configuration.md`, `specs/havn-doctor.md`, and
`specs/shared-dolt-server.md`.

## Reading Order

Use this order when moving from product framing to detailed contracts:

1. `specs/havn-overview.md` (this document) for product shape and workflow
   orientation
2. `specs/cli-framework.md` for command tree, flags, output, and CLI errors
3. `specs/configuration.md` for discovery, precedence, and effective config
4. `specs/havn-doctor.md` for diagnostic checks and output behavior
5. `specs/shared-dolt-server.md` for shared-Dolt lifecycle and safety semantics
6. `specs/environment-interface.md` for environment integration entrypoints and
   startup preparation capability boundaries
7. `specs/base-image.md` for base-image and runtime assumptions

## Runtime Model

### Project containers

Each project gets its own container. Project containers:

- bind-mount the project directory
- use the selected base image
- share the configured Nix and XDG volumes
- optionally receive shared-Dolt connectivity env vars

Runtime-level Nix registry alias persistence is backed by the mounted state
volume, so aliases survive container recreation and are shared by projects that
mount the same state volume.

Startup resource behavior at overview level:

- resource limits are sticky per project container instance
- reusing an existing running or stopped container keeps its existing limits
- recreating a missing project container applies the startup-effective resource
  config (defaulting to 4 CPUs, `8g` memory, and `12g` memory+swap when no
  overrides are provided)

### Shared Dolt mode

When enabled for a project, `havn` uses one shared `havn-dolt` container for
all project databases. Overview-level expectations:

- the server lifecycle is independent from project containers
- each project gets a separate database on the shared server
- `bd` remains the main interface for issue data inside the container

Exact lifecycle, readiness, import, export, and safety semantics are owned by
`specs/shared-dolt-server.md`.

### Doctor

`havn doctor` is the diagnostic entry point. At overview level it:

- checks host prerequisites and shared infrastructure
- checks container-level wiring for relevant running project containers
- reports problems without modifying the system

Check definitions and output details are owned by `specs/havn-doctor.md`.

## User-Facing Docs

Derivative docs in `README.md` and `docs/` should:

- describe current supported behavior
- use status labels when behavior is partial or planned
- link back to the authoritative specs for exact normative detail

This overview stays intentionally stable while the subsystem specs carry the
detailed contracts.
