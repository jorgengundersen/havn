# havn CLI reference

This is a derivative guide to the current CLI surface.

For the normative CLI contract, see `specs/cli-framework.md`.

## Status labels

- `Implemented`: supported and intended to work today
- `Partial`: user-facing surface exists, but the full spec contract is still being tightened
- `Planned`: documented direction, not shipped command behavior yet

## Global flags

These persistent flags apply to all commands:

- `--json`: machine-readable JSON output (`stdout` data; `stderr` errors)
- `--verbose`: detailed status output and diagnostics
- `--config <path>`: alternate global config file for this invocation

## Root-only runtime flags

These are accepted by `havn [path]` and are not global:

- `--shell <name>` (`HAVN_SHELL`)
- `--env <flake-ref>` (`HAVN_ENV`)
- `--cpus <n>` (`HAVN_CPUS`)
- `--memory <size>` (`HAVN_MEMORY`)
- `--port <port>` (`HAVN_SSH_PORT`)
- `--no-dolt`
- `--image <name>` (`HAVN_IMAGE`)

`havn build` may also honor `--image`, but that does not make it a persistent
flag.

`memory_swap` has no runtime flag or environment-variable override surface.
Adjust it through config files (`[resources].memory_swap`).

## Output modes and JSON conventions

- stream separation is always enforced for normal command execution: status and
  errors go to `stderr`, command data goes to `stdout`
- normal mode writes concise human-readable output
- `--verbose` adds detailed diagnostics to `stderr`
- `--json` writes structured JSON to `stdout` for query commands and
  non-interactive action command results
- root startup (`havn [path]`) retains baseline Nix diagnostics for post-run troubleshooting even in normal output mode
- action commands are completion-oriented:
  - human mode: completion/status text is written to `stderr`
  - JSON mode: a result object is written to `stdout` (status/progress may still
    appear on `stderr`)
- query commands return data only (no success wrapper) in both human and JSON
  modes
- action commands return JSON result objects in JSON mode, typically:

```json
{"status":"ok","message":"..."}
```

- JSON errors are emitted on `stderr` and include `error`; typed errors may also
  include `type` and `details`

Action success-message examples in JSON mode:

```json
{"status":"ok","message":"container running","container":"havn-user-api","project_path":"/home/user/work/api","startup_checks":"default","startup_check_phases":[]}
{"status":"ok","message":"container stopped","container":"havn-user-api"}
{"status":"ok","message":"base image built"}
```

`havn list` examples:

```text
havn-user-api	/home/user/work/api
```

```json
[
  {
    "name": "havn-user-api",
    "path": "/home/user/work/api",
    "image": "havn-base:latest",
    "status": "running",
    "shell": "bash",
    "cpus": 4,
    "memory": "8g",
    "memory_swap": "12g",
    "dolt": false
  }
]
```

For retained startup-log investigation and cleanup workflow, see `docs/doctor-troubleshooting.md`.

## Command reference

### Root command

- `havn [path]`: start or attach to the project container
- `havn --version`: print CLI version

Environment startup-check and preparation behavior:

- `havn [path]` runs startup checks plus optional environment preparation before
  attach; if preparation runs and fails, the command fails
- `havn up [path]` is non-interactive and defaults to lifecycle-only startup
  (no startup validation or preparation)
- `havn up --validate [path]` runs required startup validation, then exits
  without attaching
- `havn up --prepare [path]` runs required startup validation plus optional
  environment preparation, then exits without attaching
- `havn enter [path]` remains plain-shell entry and does not run startup
  preparation
- missing optional capability is not a startup failure
- ad-hoc `nix develop` usage from entered sessions remains supported
- for alias setup and quickstart commands inside entered sessions, see
  `docs/configuration-guide.md` (`Nix session quickstart`)

Normative behavior and scope live in `specs/cli-framework.md` and
`specs/environment-interface.md`.

The environment-interface contract itself is already ratified at
`Status: Partial`; current gaps in command behavior are runtime alignment.

Root startup resource behavior:

- `--cpus` and `--memory` apply when creating a new project container
- reusing an existing running or stopped container keeps its existing limits
- defaults on create are `cpus=4`, `memory=8g`, `memory_swap=12g` when no
  override is supplied
- to apply changed limits to an existing project, remove that project container
  and run startup again

### Core commands

- `havn up [path]`: run lifecycle startup without interactive attach
- `havn enter [path]`: enter a running project container with plain `bash`
- `havn list`: list running havn-managed containers as `name<TAB>path` (or JSON)
- `havn stop [name|path]`: stop one project container
- `havn stop --all`: stop all running havn-managed containers with
  best-effort reporting
- `havn build`: build the base image used for project containers

Startup-check examples for `havn up`:

```bash
havn up .
havn up --validate .
havn up --prepare .
havn enter .
```

`havn stop [name|path]` target rules:

- path-like targets (`.`, `..`, values containing path separators, and absolute
  paths) are resolved as paths
- relative path-like targets are valid (for example `havn stop .`)
- path-like targets must resolve to an existing directory
- non-path-like targets are treated as literal container names
- invalid path-like targets return path-related errors rather than
  container-name-not-found errors

### Grouped commands

- `havn config show`: inspect effective merged configuration
- `havn volume list`: inspect configured shared volume presence
- `havn doctor`: run host and container health checks

### Dolt commands

- `havn dolt start`
- `havn dolt stop`
- `havn dolt status`
- `havn dolt databases`
- `havn dolt drop <name> --yes`
- `havn dolt connect`
- `havn dolt import <path> [--force]`
- `havn dolt export <name> [--dest <path>]`

Ownership boundary for migration surfaces:

- `havn` owns shared-Dolt infrastructure lifecycle and command framing
- migration correctness, project-identity migration policy, rollback, and
  reconciliation semantics are owned by beads/Dolt workflows
- for migration-policy expectations, follow beads tooling/contracts

### Planned utility commands

- `havn completion <bash|zsh|fish|powershell>`: planned, not part of the
  shipped command tree today

## Support matrix

| Command | Status | Notes |
|---|---|---|
| `havn [path]` | Implemented | Start-or-attach entry point |
| `havn up [path]` | Implemented | Lifecycle startup without attach; contract owned by `specs/cli-framework.md` |
| `havn enter [path]` | Implemented | Plain `bash` entry for running project containers |
| `havn list` | Implemented | Query semantics: human `name<TAB>path`; JSON array of container records |
| `havn stop` | Implemented | Single stop and `--all` best-effort behavior |
| `havn build` | Implemented | Base-image build surface |
| `havn config show` | Partial | Normative config contract lives in `specs/configuration.md` |
| `havn volume list` | Implemented | Shared volume inspection |
| `havn doctor` | Implemented | Normative doctor contract lives in `specs/havn-doctor.md` |
| `havn dolt start/stop/status/databases/drop/connect/import/export` | Implemented | Normative Dolt contract lives in `specs/shared-dolt-server.md` |
| `havn completion` | Planned | Planned CLI surface owned by `specs/cli-framework.md` |

## Current partial-support gaps

- `havn config show` currently publishes source provenance for core scalar/resource/Dolt fields, but not for every effective-config field in the output

When this guide and a spec disagree, follow the relevant spec in `specs/`.
