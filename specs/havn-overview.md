# havn Overview

This document is the product overview for `havn`.

Status: Partial

It explains product shape, primary workflows, and reading order. It is not the
authoritative owner for detailed configuration, CLI, doctor, or shared-Dolt
contracts.

## Product Shape

`havn` is a Go CLI for reproducible development environments built on Docker and
Nix. It manages one container per project, reuses shared host-side
infrastructure where possible, and attaches the user directly into the selected
dev shell.

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
- Doctor checks, tiering, selection rules, and doctor output:
  `specs/havn-doctor.md`
- Shared Dolt lifecycle, readiness, ownership, migration, and safety semantics:
  `specs/shared-dolt-server.md`
- Base image and runtime assumptions: `specs/base-image.md`

## Core Workflow

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

On successful attach, the root command exits with the shell session's exit code.

Detailed config resolution lives in `specs/configuration.md`. Detailed CLI and
error behavior lives in `specs/cli-framework.md`.

### Shared infrastructure

Startup may create or reuse:

- the configured Docker network
- named Docker volumes for Nix and XDG state
- the configured base image
- the shared Dolt server when project config enables it

These resources are shared across projects. They are not torn down as part of a
single project startup failure.

### Stop behavior

- `havn stop <name|path>` stops one project container
- `havn stop --all` stops all running havn-managed project containers using
  best-effort semantics
- shared Dolt lifecycle is separate from project-container stop behavior

The detailed stop contract and output semantics live in
`specs/cli-framework.md` and `specs/shared-dolt-server.md`.

## Command Map

This is a product map, not the detailed CLI contract.

| Surface | Status | Authority |
|------|------|------|
| `havn [path]` | Implemented | `specs/cli-framework.md` |
| `havn list` | Implemented | `specs/cli-framework.md` |
| `havn stop` | Implemented | `specs/cli-framework.md` |
| `havn build` | Implemented | `specs/cli-framework.md` + `specs/base-image.md` |
| `havn config show` | Implemented, contract still tightening | `specs/configuration.md` |
| `havn volume list` | Implemented | `specs/cli-framework.md` |
| `havn doctor` | Implemented, contract still tightening | `specs/havn-doctor.md` |
| `havn dolt ...` | Implemented, contract still tightening | `specs/shared-dolt-server.md` |
| `havn completion ...` | Planned | `specs/cli-framework.md` |

## Runtime Model

### Project containers

Each project gets its own container. Project containers:

- bind-mount the project directory
- use the selected base image
- share the configured Nix and XDG volumes
- optionally receive shared-Dolt connectivity env vars

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
