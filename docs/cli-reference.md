# havn CLI reference

This reference documents the current `havn` CLI surface as implemented today and highlights where behavior is still planned.

## Global flags

These flags apply to all commands:

- `--json`: machine-readable JSON output
- `--verbose`: detailed status output, command-level diagnostics
- `--config <path>`: explicit global config file path

Root-only runtime flags (for `havn [path]` and `havn build` image resolution) are not global:

- `--shell <name>` (`HAVN_SHELL`)
- `--env <flake-ref>` (`HAVN_ENV`)
- `--cpus <n>` (`HAVN_CPUS`)
- `--memory <size>` (`HAVN_MEMORY`)
- `--port <port>` (`HAVN_SSH_PORT`)
- `--no-dolt`
- `--image <name>` (`HAVN_IMAGE`)

## Output modes and JSON conventions

- Stream separation is always enforced: status/errors to `stderr`, command data to `stdout`.
- Normal mode writes concise human-readable output.
- `--verbose` adds detailed diagnostics to `stderr`.
- `--json` writes structured JSON to `stdout` for data-producing commands.
- Action commands return JSON result objects in JSON mode, typically:

```json
{"status":"ok","message":"..."}
```

- JSON errors are emitted on `stderr` and include `error`; typed errors may include `type` and `details`.

## Command reference

### Root command

- `havn [path]`: start or attach to the project container (path defaults to `.`)
- `havn --version`: print CLI version

### Core commands

- `havn list`: list running havn-managed project containers
- `havn stop [name|path]`: stop one project container
- `havn stop --all`: stop all running havn-managed project containers (best effort)
- `havn build`: build the base image used for project containers

### Grouped commands

- `havn config show`: show effective merged configuration
- `havn volume list`: list managed/shared volume presence
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

### Utility commands

- `havn completion <bash|zsh|fish|powershell>`

## Support matrix

The table below describes implementation status in the current CLI.

| Command | Status | Notes |
|---|---|---|
| `havn [path]` | Implemented | Start/attach flow wired through container lifecycle service |
| `havn list` | Implemented | Human + JSON output |
| `havn stop` | Implemented | Single stop and `--all` best-effort behavior |
| `havn build` | Implemented | Builds configured base image |
| `havn config show` | Implemented | Includes JSON `source` metadata object |
| `havn volume list` | Implemented | Reports configured volumes and existence |
| `havn doctor` | Implemented | Exit codes: 0 pass, 1 warn, 2 error |
| `havn dolt start/stop/status/databases/drop/connect/import/export` | Implemented | Shared-server mode |
| `havn completion` | Implemented | Cobra built-in completion generator |
| Additional commands from future specs | Planned | Not yet part of the current command tree |
