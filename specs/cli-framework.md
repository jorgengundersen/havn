# CLI Framework

This is the authoritative CLI contract for `havn`.

Status: Partial

`Partial` means this document owns the intended CLI behavior, but some commands
or support claims may not yet be fully implemented.

## Ownership

This spec owns:

- the command tree
- flag scope terminology
- stream separation and output modes
- startup logging mode boundaries for root startup
- session-entry lifecycle boundaries for Home Manager activation
- JSON contract ownership at the CLI boundary
- CLI error formatting and exit-code rules

Configuration precedence and effective-config semantics are owned by
`specs/configuration.md`.

## Command Tree

The command tree is:

```text
havn
  up
  enter
  list
  stop
  build
  config
    show
  volume
    list
  doctor
  dolt
    start
    stop
    status
    databases
    drop
    connect
    import
    export
  completion
```

### Command-group rules

- `config`, `volume`, and `dolt` are parent commands used only for namespacing.
- Parent commands without subcommands print help rather than performing work.
- The root command is the only command with a default action.

### Support status

| Surface | Status |
|------|------|
| `havn [path]` | Implemented |
| `havn up [path]` | Implemented |
| `havn enter [path]` | Implemented |
| `havn list` | Implemented |
| `havn stop` | Implemented |
| `havn build` | Implemented |
| `havn config show` | Partial |
| `havn volume list` | Implemented |
| `havn doctor` | Partial |
| `havn dolt start/stop/status/databases/drop/connect/import/export` | Partial |
| `havn completion <shell>` | Planned |

Derivative docs must not label a command as implemented when its published
contract is still marked partial or planned here.

## Flag Scope Vocabulary

### Persistent flags

Persistent flags are defined on the root command and inherited by subcommands.

`havn` persistent flags are:

- `--json`
- `--verbose`
- `--config <path>`

These are the only global flags.

### Startup runtime flags

Startup runtime flags are non-persistent flags used by startup-oriented command
surfaces.

Shared startup runtime flags (accepted by `havn [path]` and
`havn up [path]`) are:

- `--env <flake-ref>`
- `--cpus <n>`
- `--memory <size>`
- `--port <port>`
- `--no-dolt`
- `--image <name>`

Attach-only startup runtime flag:

- `--shell <name>` (accepted by `havn [path]` only)

`havn up [path]` must not accept `--shell` because `up` does not attach
to an interactive shell session.

`havn build` may also honor `--image` because build-time image selection is part
of its own contract, but that does not make `--image` a persistent flag.

### Command-local flags

Command-local flags apply only to one command.

Examples:

- `havn stop --all`
- `havn doctor --all`
- `havn dolt drop --yes`
- `havn dolt import --force`
- `havn dolt export --dest <path>`

## Root Command Contract

Usage:

```text
havn [flags] [path]
```

- `path` is optional and defaults to `.`
- successful interactive attach exits with the shell session's exit code
- startup failure exits through normal CLI error handling

For startup runtime resource flags (`--cpus`, `--memory`):

- values apply when creating a new project container for the resolved path
- values do not retroactively mutate an existing project container that is
  being reused (running or stopped)
- applied limits for a newly created container must be visible at create time in
  container metadata/inspection surfaces

`havn [path]` and `havn up [path]` are implemented startup entry points.
`havn up [path]` shares startup runtime flags except `--shell`.

### Startup and entry workflow split

`havn` has three workflow surfaces with distinct intent:

- `havn [path]`: start-or-attach, then enter `nix develop <ref>#<shell> -c bash`
- `havn up [path]`: lifecycle startup without interactive attach
- `havn enter [path]`: interactive plain `bash` entry without `nix develop`

`havn [path]`, `havn up [path]`, and `havn enter [path]` are implemented.

`havn enter [path]` returns an actionable CLI error for missing or stopped
project containers that includes `havn up <path>` guidance.

Before plain-shell attach, `havn enter [path]` performs the same in-container
Nix registry persistence preparation as startup-oriented entry, so users do not
need to run startup first solely for registry alias persistence.

### Home Manager session lifecycle (Planned)

Home Manager integration is a planned extension of the startup/entry split.

- `havn [path]` is the primary interactive flow and should perform Home Manager
  user-configuration activation before presenting the session shell
- `havn up [path]` stays non-interactive and should not attach or prompt; any
  Home Manager activation tied to `up` must preserve that non-interactive
  contract
- `havn enter [path]` remains plain-shell entry and should offer a documented
  manual activation path rather than silently changing plain-entry semantics
- ad-hoc `nix develop` usage from an entered session remains valid and must not
  be blocked by Home Manager lifecycle orchestration

### Startup logging contract

For root startup (`havn [path]`), logging behavior is:

- baseline startup diagnostics are retained by default for post-run investigation
- default terminal UX stays concise (status-focused)
- `--verbose` is an opt-in startup mode that streams detailed diagnostics to
  `stderr` during startup

Verbose startup is intentionally flag-only. No config key or environment
variable changes startup diagnostic verbosity.

## Output Contract

### Stream separation

This is an invariant across CLI commands:

- `stderr`: status messages, progress, logs, and errors
- `stdout`: command data and machine-readable JSON payloads

Interactive shell attach is the one special case: Docker TTY mode is an
interactive stream, so `havn` does not promise separate stderr capture during
the attached shell session.

### Output modes

| Mode | Trigger | `stderr` | `stdout` |
|------|------|------|------|
| normal | default | concise status | human-readable data |
| verbose | `--verbose` | status plus detailed diagnostics | human-readable data |
| json | `--json` | status unchanged | structured JSON data |

`--verbose --json` is valid. Verbose diagnostics stay on `stderr`; JSON data is
written to `stdout`.

For root startup, retained baseline diagnostics are independent of output mode:
normal, verbose, and json all keep retained startup diagnostics for later
investigation.

### JSON ownership

The CLI boundary owns the exact JSON emitted by commands. Domain packages return
domain values; the CLI layer formats them into the stable command JSON shape.

- data JSON is written to `stdout`
- errors in JSON mode are written to `stderr`
- action-only commands return a JSON result object when `--json` is active

Typical action result shape:

```json
{"status":"ok","message":"..."}
```

Field additions are non-breaking. Field removals or renames are breaking.

## Error Handling

### `RunE` boundary

Commands use `RunE` so errors propagate to the root execution boundary.

### Human-readable errors

In normal mode, CLI errors are printed to `stderr` as actionable messages.

### JSON errors

In JSON mode, errors are written to `stderr` as JSON.

When a typed error is available, the JSON includes:

- `error`
- `type`
- optional `details`

Fallback shape:

```json
{"error":"container \"havn-user-api\" not found"}
```

### Exit codes

Default exit codes:

- `0`: success
- `1`: command error

Command-specific exit codes may extend this. `havn doctor` is the main example:

- `0`: all checks passed
- `1`: warnings present
- `2`: errors present

## Command Notes

### `havn config show`

- produces the effective-config inspection output
- output contract is owned jointly with `specs/configuration.md`
- source and provenance semantics are owned by `specs/configuration.md`

### `havn doctor`

- check definitions and selection semantics are owned by `specs/havn-doctor.md`
- this spec still owns shared CLI rules such as `--json`, `--verbose`, stream
  separation, and exit-code handling

### `havn dolt ...`

- shared-Dolt lifecycle, readiness, ownership, and safety semantics are owned by
  `specs/shared-dolt-server.md`
- this spec owns the command naming and CLI-level flag and output rules

### `havn completion`

`havn completion <bash|zsh|fish|powershell>` is a planned command surface. When
it lands, it should expose Cobra-generated completions unless a later spec
revision defines custom completion behavior.

## Testing Expectations For The CLI Layer

CLI tests should verify:

- flag parsing and scope
- argument handling
- output routing to `stdout` vs `stderr`
- JSON and human output selection
- error formatting and exit-code behavior
- command registration and reachability

CLI tests should not retest domain behavior that belongs below the CLI layer.
