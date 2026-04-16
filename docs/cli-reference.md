# havn CLI reference

This is a derivative guide to the current CLI surface.

For the normative CLI contract, see `specs/cli-framework.md`.

## Status labels

- `Implemented`: supported and intended to work today
- `Partial`: user-facing surface exists, but the full spec contract is still being tightened
- `Planned`: documented direction, not shipped command behavior yet

## Global flags

These persistent flags apply to all commands:

- `--json`: machine-readable JSON output
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

## Output modes and JSON conventions

- stream separation is always enforced for normal command execution: status and
  errors go to `stderr`, command data goes to `stdout`
- normal mode writes concise human-readable output
- `--verbose` adds detailed diagnostics to `stderr`
- `--json` writes structured JSON to `stdout` for data-producing commands
- root startup (`havn [path]`) retains baseline Nix diagnostics for post-run troubleshooting even in normal output mode
- action commands return JSON result objects in JSON mode, typically:

```json
{"status":"ok","message":"..."}
```

- JSON errors are emitted on `stderr` and include `error`; typed errors may also
  include `type` and `details`

For retained startup-log investigation and cleanup workflow, see `docs/doctor-troubleshooting.md`.

## Command reference

### Root command

- `havn [path]`: start or attach to the project container
- `havn --version`: print CLI version

### Core commands

- `havn list`: list havn-managed project containers
- `havn stop [name|path]`: stop one project container
- `havn stop --all`: stop all running havn-managed project containers with
  best-effort reporting
- `havn build`: build the base image used for project containers

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

### Planned utility commands

- `havn completion <bash|zsh|fish|powershell>`: planned, not part of the
  shipped command tree today

## Support matrix

| Command | Status | Notes |
|---|---|---|
| `havn [path]` | Implemented | Start-or-attach entry point |
| `havn list` | Implemented | Human and JSON output |
| `havn stop` | Implemented | Single stop and `--all` best-effort behavior |
| `havn build` | Implemented | Base-image build surface |
| `havn config show` | Partial | Normative config contract lives in `specs/configuration.md` |
| `havn volume list` | Implemented | Shared volume inspection |
| `havn doctor` | Partial | Normative doctor contract lives in `specs/havn-doctor.md` |
| `havn dolt start/stop/status/databases/drop/connect/import/export` | Partial | Normative Dolt contract lives in `specs/shared-dolt-server.md` |
| `havn completion` | Planned | Planned CLI surface owned by `specs/cli-framework.md` |

## Current partial-support gaps

- `havn config show` currently publishes source provenance for core scalar/resource/Dolt fields, but not for every effective-config field in the output

When this guide and a spec disagree, follow the relevant spec in `specs/`.
