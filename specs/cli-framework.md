# CLI Framework

This is the authoritative CLI contract for `havn`.

Status: Partial

`Partial` means this document owns the intended CLI behavior, but some commands
or support claims may not yet be fully implemented.

## Ownership

This spec owns:

- the command tree
- root utility flag behavior including `havn --version`
- flag scope terminology
- stream separation and output modes
- startup logging mode boundaries for root startup
- session-entry lifecycle boundaries for optional environment startup preparation
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
| `havn doctor` | Implemented |
| `havn dolt start/stop/status/databases/drop/connect` | Implemented |
| `havn dolt import/export` | Implemented (command execution framing only; migration semantics are a non-goal) |
| `havn completion <shell>` | Implemented |

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

`havn up [path]` startup-check flags:

- `--validate`
- `--prepare` (implies `--validate`)

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

- `havn --version` prints the CLI version string and exits `0` without
  running startup or attach behavior
- `path` is optional and defaults to `.`
- successful interactive attach exits with the shell session's exit code
- startup failure exits through normal CLI error handling

Startup project-path boundary ownership is delegated to
`specs/configuration.md`: for `havn [path]` and `havn up [path]`, whether the
resolved project path is valid under the user's home directory is defined there.

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

- `havn [path]`: start-or-attach, then enter interactive `nix develop <ref>#<shell>`
- `havn up [path]`: lifecycle startup without interactive attach; default run is
  container lifecycle only
- `havn enter [path]`: interactive plain `bash` entry without `nix develop`

`havn [path]`, `havn up [path]`, and `havn enter [path]` are implemented.

`havn enter [path]` returns an actionable CLI error for missing or stopped
project containers that includes `havn up <path>` guidance.

Before plain-shell attach, `havn enter [path]` performs in-container
Nix registry persistence preparation, so users do not need startup checks to
run first solely for registry alias persistence.

### Environment startup preparation lifecycle contract

Startup preparation is environment-owned and capability-driven. The capability
entrypoint is defined by `specs/environment-interface.md`.

The environment-interface contract is ratified at `Status: Partial`; remaining
gaps for this behavior are runtime-alignment work, not contract-planning work.

- `havn [path]` runs required environment validation and then runs the optional
  prepare capability when present before shell handoff. If either phase fails,
  `havn` exits non-zero and must not attach to an interactive shell.
- `havn up [path]` by default does not run environment validation or optional
  prepare capability; it remains non-interactive and lifecycle-focused.
- default `havn up [path]` lifecycle startup also does not run in-container Nix
  registry persistence preparation.
- `havn up [path] --validate` runs required environment validation and remains
  non-interactive.
- `havn up [path] --prepare` runs required environment validation and then runs
  optional prepare capability when present; it remains non-interactive.
- for `havn up [path]` with `--validate` or `--prepare`, phase failures exit
  non-zero with actionable command-scoped guidance.
- `havn enter [path]` remains plain-shell entry and does not run startup
  preparation.
- Missing optional capability is not a command failure.
- Startup preparation behavior must not invalidate ad-hoc `nix develop` usage
  from entered sessions.

### Startup phase observability contract

For startup checks and preparation phases:

- phase start and completion events are emitted on `stderr`
- completion events include elapsed duration and remain concise in default mode
- streaming stderr lines from startup-check commands are classified into compact
  activity updates (`evaluate`, `fetch`, `build`, or `other`) and emitted as
  phase progress status lines
- when progress lines include counters/rates, compact parsed metrics are
  appended to the status line (for example item counts, byte totals, transfer
  rate)
- long-running phases emit progress heartbeat messages on `stderr` at least
  every 10 seconds while the phase is active
- heartbeat updates include the last classified progress activity when available
- `--verbose` includes concrete command details for active startup phases
- interrupting startup reports the interrupted phase before exiting
- phase-scoped failures report the phase name, exit semantics, and actionable
  guidance

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

This is an invariant for non-interactive command output across CLI commands:

- `stderr`: status messages, progress, logs, and errors
- `stdout`: command data and machine-readable JSON payloads

Interactive attach is the explicit exception boundary. While an attached
subprocess owns the TTY stream (`havn [path]`, `havn enter [path]`, `havn dolt
connect`), `havn` does not promise separate stderr capture for that interactive
session.

Outside that attached session window (startup/status output before attach and
error reporting after attach exits), normal stream-separation rules still
apply.

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
- action-only commands that are non-interactive return a JSON result object when
  `--json` is active

### Cross-command success/result semantics

Output behavior is contract-driven by command type.

| Command type | Human mode | JSON mode |
|------|------|------|
| action command | status/completion on `stderr`; no data payload on `stdout` | result object on `stdout`; status/progress may still appear on `stderr` |
| query command | data result on `stdout`; no completion wrapper | structured result on `stdout`; no success wrapper |
| interactive command | interactive stream owned by subprocess/TTY | same interactive behavior; no synthetic success payload |

### Command-type mapping (ratified)

| Command surface | Type | JSON success/result shape contract |
|------|------|------|
| `havn [path]` | interactive | no success payload contract; shell/session stream is interactive |
| `havn up [path]` | action | object with `status`, `message`, `container`, `project_path`, `startup_checks`, `startup_check_phases` |
| `havn enter [path]` | interactive | no success payload contract; shell/session stream is interactive |
| `havn stop [name|path]` | action | object with `status`, `message`; single-target also includes `container` |
| `havn stop --all` | action (best-effort) | object with `status`, `message`; `status` is `"error"` when any stop fails |
| `havn list` | query | array of container records |
| `havn build` | action | object with `status`, `message` |
| `havn doctor` | query | doctor report object |
| `havn config show` | query | effective-config object |
| `havn volume list` | query | array of volume-entry records |
| `havn dolt start/stop/drop/import/export` | action | object with `status`, `message`, plus command-specific fields |
| `havn dolt status` | query | status object |
| `havn dolt databases` | query | array of database names |
| `havn dolt connect` | interactive | no success payload contract; SQL shell session is interactive |

### JSON field naming and presence rules

- key names are `snake_case`
- action result objects require `status` and `message`
- `status` values are stable and command-scoped:
  - `"ok"` for successful completion
  - `"error"` only for best-effort mixed outcomes where a summary payload is
    still returned (for example `havn stop --all` with failures)
- command identity fields (`container`, `project_path`, `database`, `dest`,
  `startup_checks`, `ownership_boundary`, and similar command-local identity
  keys) must be stable once published
- fields are omitted when not applicable; commands must not emit placeholder
  keys with empty semantic value solely for shape parity
- list fields emit arrays (`[]`) rather than `null` when no items are present

Field additions are non-breaking. Field removals or renames are breaking.

## Error Handling

### `RunE` boundary

Commands use `RunE` so errors propagate to the root execution boundary.

### Human-readable errors

In normal mode, CLI errors are printed to `stderr` as actionable messages.

### JSON errors

In JSON mode, errors are written to `stderr` as JSON.

Error framing is command-scoped. Command handlers return errors wrapped with the
command surface (`havn up`, `havn stop`, `havn dolt start`, and so on) so users
and automation can attribute failure to the invoked command.

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

### `havn stop [name|path]`

- accepts exactly one positional target unless `--all` is set
- target parsing is shape-based:
  - path-like targets (`.`, `..`, values containing path separators, and
    absolute paths) are treated as filesystem paths
  - non-path-like targets are treated as literal container names
- path-like targets are resolved to canonical absolute paths and must point to
  an existing directory
- for path-like targets, container names are derived from the canonical path
  using the same project-name derivation rules as startup commands
- invalid path-like targets fail with actionable path errors (for example,
  missing path or not-a-directory), not container-name lookup errors
- single-target success status reports the resolved container identity, never
  the raw CLI target text
- in human mode, single-target success output is `Stopped <container-name>`
  for both literal-name and path-like inputs
- in JSON mode, single-target success output includes the resolved container
  identity as a stable `container` field in addition to baseline success fields
- for path-like targets (for example `.` and `./project`), the reported
  container identity is the derived havn container name

### `havn up [path]`

- in human mode, successful runs report lifecycle completion on `stderr`
- in JSON mode, successful runs emit a stable action result object on `stdout`
  with:
  - `status` (`"ok"`)
  - `message` (`"container running"`)
  - `container` (resolved container name)
  - `project_path` (resolved project path)
  - `startup_checks` (`"default"`, `"validate"`, or `"prepare"`)
  - `startup_check_phases` (array of completed startup-check phase summaries,
    each with `phase`, `outcome`, and `duration_ms`)
- startup-check failures (`--validate` / `--prepare`) are command-scoped in
  error framing and exit non-zero
- in JSON mode, startup-check errors are emitted on `stderr` via the shared
  JSON error contract

### `havn config show`

- produces the effective-config inspection output
- output contract is owned jointly with `specs/configuration.md`
- source and provenance semantics are owned by `specs/configuration.md`

### `havn doctor`

- check definitions and selection semantics are owned by `specs/havn-doctor.md`
- this spec still owns shared CLI rules such as `--json`, `--verbose`, stream
  separation, and exit-code handling

### `havn dolt ...`

- shared-Dolt lifecycle, readiness, ownership, startup provisioning, and
  status/databases semantics are owned by `specs/shared-dolt-server.md`
- beads data-migration and identity policy semantics are outside this spec's
  ownership boundary
- this spec owns the command naming and CLI-level flag and output rules
- for `havn dolt import/export`, CLI ownership is limited to command execution
  framing (status/error/result shape), not migration correctness policy

`havn dolt import/export` migration correctness semantics are an explicit
non-goal for CLI ownership and remain owned by beads/Dolt workflows.

### `havn completion`

`havn completion <bash|zsh|fish|powershell>` is implemented and exposes
Cobra-generated completion scripts.

## Testing Expectations For The CLI Layer

CLI tests should verify:

- flag parsing and scope
- argument handling
- output routing to `stdout` vs `stderr`
- JSON and human output selection
- error formatting and exit-code behavior
- command registration and reachability

CLI tests should not retest domain behavior that belongs below the CLI layer.
