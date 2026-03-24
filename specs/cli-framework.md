# CLI Framework

How havn's CLI is structured, wired, and tested. Implements the CLI layer
described in [architecture-principles.md](architecture-principles.md) Section 1
and the interface defined in [havn-overview.md](havn-overview.md).

Several specs forward-reference this one:

- Architecture principles: _"Command structure, flag naming, and UX conventions
  live in a dedicated CLI spec."_ (Section 1)
- Architecture principles: _"Logging levels, output formats, and instrumentation
  patterns in implementation specs."_ (Section 9)
- Code standards: _"`cmd/havn/` for the entry point. Wiring only."_ (Section 1)

---

## 1. Framework Choice: Cobra

_Ref: [Principles 3](architecture-principles.md) (build vs depend)_

havn uses [spf13/cobra](https://github.com/spf13/cobra) for CLI parsing.

**Why depend rather than build:**

- Cobra is the de facto standard for Go CLIs. Its conventions (subcommands,
  persistent flags, help generation) are familiar to Go developers and AI
  agents alike. Building our own would reimplement the same patterns with
  less battle-testing.
- The surface havn uses is large: nested subcommands (`havn dolt start`),
  persistent vs local flags, positional arguments, automatic help/usage
  generation, shell completions. This is not a thin wrapper -- implementing
  it correctly is non-trivial.
- Cobra is a single dependency with minimal transitive cost (`pflag` only).
  The supply chain risk is low relative to the implementation cost saved.
- Correctness matters: flag parsing, argument validation, and help formatting
  have subtle edge cases. Cobra has years of these solved.

**What we do NOT use from Cobra:**

- `cobra-cli` scaffolding tool. Commands are written by hand.
- `viper` for configuration. Config loading is a separate domain concern
  ([havn-overview.md](havn-overview.md) configuration section) with its own
  package. Cobra handles CLI flags only; config merging happens in domain code.

## 2. Command Tree

_Ref: [havn-overview.md](havn-overview.md) subcommands table_

The command tree maps directly to the subcommand table in the overview spec:

```
havn                          root command (start/attach)
  list                        list running containers
  stop                        stop container(s)
  build                       build base image
  config
    show                      show effective merged config
  volume
    list                      list havn volumes
  doctor                      diagnose environment health
  dolt
    start                     start shared Dolt server
    stop                      stop shared Dolt server
    status                    show Dolt server status
    databases                 list databases
    drop                      drop a project database
    connect                   open SQL shell
    import                    import local database
    export                    export database
```

### Command grouping

`config`, `volume`, and `dolt` are **parent commands** that exist only to
namespace their subcommands. They have no `RunE` of their own -- invoking
`havn config` without a subcommand prints help. This is standard Cobra
behavior for command groups.

### Root command behavior

The root command (`havn [flags] [path]`) is the only command with a default
action. It accepts an optional positional argument (project path, defaults
to `.`). All other commands use explicit subcommand names with no positional
ambiguity.

## 3. Package Layout

_Ref: [code-standards.md](code-standards.md) Section 1_

```
cmd/havn/
  main.go           program entry point -- wiring only
internal/
  cli/
    root.go          root command definition and execution
    root_test.go
    list.go          havn list
    list_test.go
    stop.go          havn stop
    stop_test.go
    build.go         havn build
    build_test.go
    config.go        havn config show
    config_test.go
    volume.go        havn volume list
    volume_test.go
    doctor.go        havn doctor
    doctor_test.go
    dolt.go          havn dolt *
    dolt_test.go
    output.go        output mode helpers (normal, verbose, JSON)
    output_test.go
    errors.go        CLI-layer error formatting
    errors_test.go
```

### `cmd/havn/main.go`

Minimal. Calls `cli.Execute()` and exits. No logic beyond the call:

```go
package main

import (
    "os"

    "github.com/jorgengundersen/havn/internal/cli"
)

func main() {
    os.Exit(cli.Execute())
}
```

`Execute()` is a convenience that wires real dependencies and runs the
command tree. It constructs production implementations, passes them to
`NewRoot()`, and translates the result to an exit code. See
[Section 9](#9-dependency-injection-for-commands) for the full wiring
pattern.

Tests bypass `Execute()` entirely and call `NewRoot(fakeDeps)` directly,
giving them full control over injected dependencies.

### `internal/cli/`

All Cobra command definitions live here. Each file defines one command (or
a command group like `dolt.go` for the dolt subcommands). This package is
the CLI boundary -- where user input enters and user-facing output leaves.

**Commands are thin.** A command function:

1. Reads flags and args from Cobra.
2. Calls domain code (packages under `internal/`).
3. Formats the result for output (respecting output mode).
4. Returns errors as user-facing messages.

Domain logic does not live in `internal/cli/`. If a command function is
growing beyond wiring, the logic belongs in a domain package.

### Adding a new command

When a new feature lands, adding its CLI surface follows this pattern:

1. Create `internal/cli/<command>.go` with the Cobra command definition.
2. Register it in the parent command's `init()` or setup function.
3. Create `internal/cli/<command>_test.go` with CLI-level tests.
4. The command calls into the relevant domain package under `internal/`.

The skeleton is wired first with a stub implementation that returns a
"not implemented" error. The domain package is built separately (TDD),
then the stub is replaced with the real call. This ensures the CLI
surface is testable before the domain logic exists.

## 4. Flag Handling

_Ref: [havn-overview.md](havn-overview.md) global flags table_

### Persistent flags (global)

Flags that apply broadly are defined on the root command as **persistent
flags**, making them available to all subcommands:

```go
root.PersistentFlags().BoolVar(&opts.JSON, "json", false, "machine-readable JSON output")
root.PersistentFlags().BoolVar(&opts.Verbose, "verbose", false, "show detailed output")
root.PersistentFlags().StringVar(&opts.Config, "config", "", "path to config file")
```

### Container flags (root command only)

Flags that control container behavior are local to the root command, since
it is the only command that starts containers:

```go
root.Flags().StringVar(&opts.Shell, "shell", "", "devShell to activate")
root.Flags().StringVar(&opts.Env, "env", "", "Nix flake ref for dev environment")
root.Flags().IntVar(&opts.CPUs, "cpus", 0, "CPU limit")
root.Flags().StringVar(&opts.Memory, "memory", "", "memory limit")
root.Flags().StringVar(&opts.Port, "port", "", "SSH port mapping")
root.Flags().BoolVar(&opts.NoDolt, "no-dolt", false, "skip Dolt server")
root.Flags().StringVar(&opts.Image, "image", "", "override base image")
```

This keeps help output clean -- `havn dolt status --help` does not show
`--shell` or `--cpus`, which would be confusing. Only flags that are
genuinely global (`--json`, `--verbose`, `--config`) are persistent.

### Local flags (command-specific)

Flags that apply to a single command use local flags:

```go
stopCmd.Flags().BoolVar(&stopOpts.All, "all", false, "stop all havn containers")
doltDropCmd.Flags().BoolVar(&dropOpts.Yes, "yes", false, "confirm database drop")
```

### Flag vs config precedence

Cobra handles flag parsing only. The precedence chain
(**flag > env > project config > global config > default**) is resolved in
domain code, not in Cobra. The CLI layer passes flag values and a "was this
flag explicitly set?" signal to the config merging logic:

```go
// CLI layer determines which flags were explicitly set
flagOverrides := config.Overrides{}
if cmd.Flags().Changed("shell") {
    flagOverrides.Shell = &opts.Shell
}
if cmd.Flags().Changed("cpus") {
    flagOverrides.CPUs = &opts.CPUs
}

// Domain code resolves precedence
cfg := config.Resolve(globalCfg, projectCfg, envOverrides, flagOverrides)
```

The `Changed()` check is critical -- it distinguishes "user passed
`--cpus 0`" from "user didn't pass `--cpus`". Without it, zero values from
unset flags silently override config file values.

### Environment variable bridging

Cobra does not read environment variables. Env var resolution
(`HAVN_SHELL`, `HAVN_CPUS`, etc.) is handled in domain config code, not
in the CLI layer. The CLI layer passes raw flag values; the config layer
checks env vars at its own precedence level.

This keeps Cobra's responsibility narrow (flag parsing only) and keeps
precedence logic in one place (config package).

## 5. Output Modes

_Ref: [havn-overview.md](havn-overview.md) output modes,
[code-standards.md](code-standards.md) Section 5_

### Stream separation

This is an invariant, not a mode:

- **stderr**: status messages, progress, logs, errors. Always.
- **stdout**: data output. Only when the command produces data.

Every command respects this regardless of output mode. A command that only
performs an action (e.g., `havn stop`) writes status to stderr and nothing
to stdout in normal mode.

### The three modes

| Mode | Activated by | stderr | stdout |
|------|-------------|--------|--------|
| Normal | default | minimal status lines | human-readable data |
| Verbose | `--verbose` | status + commands + timing | human-readable data |
| JSON | `--json` | status lines (unchanged) | JSON data |

`--verbose` and `--json` are independent. `--verbose --json` shows verbose
status on stderr and JSON data on stdout.

### Output abstraction

Commands do not write directly to stdout/stderr. They use an output helper
that encapsulates the current mode and enforces stream separation.

The output helper is constructed once from the global flags and passed to
every command. Its responsibilities:

- **Status messages** go to stderr, in all modes.
- **Data output** goes to stdout -- as JSON when `--json` is active,
  human-formatted otherwise.
- **Action results** (for commands that only perform an action like
  `havn stop`) write a JSON result object to stdout in JSON mode. In
  normal mode, the status message on stderr is sufficient.

The exact API of the output helper will emerge from implementation (TDD).
The constraints are: stream separation is enforced structurally (commands
cannot accidentally write status to stdout), and the mode decision happens
in one place (not scattered across command functions).

### JSON contract

JSON output shapes are defined in the overview spec per command. The CLI
layer is responsible for marshalling domain types into the documented JSON
shapes. Domain code returns domain types; the CLI layer formats them.

JSON output must be stable. Field additions are non-breaking. Field
removals or renames are breaking changes. Treat JSON output as a public
API.

## 6. Error Handling at the CLI Boundary

_Ref: [code-standards.md](code-standards.md) Section 2,
[architecture-principles.md](architecture-principles.md) Section 5_

### Cobra's RunE pattern

All commands use `RunE` (not `Run`) so errors propagate to the root:

```go
var listCmd = &cobra.Command{
    Use:   "list",
    Short: "List running havn containers",
    RunE:  runList,
}

func runList(cmd *cobra.Command, args []string) error {
    // ...
}
```

### Error formatting

`Execute()` is the single error handling boundary. Domain errors bubble
up through `RunE`; `Execute` translates them to user-facing output and
an exit code.

**Normal mode:** errors are printed to stderr as actionable messages.

```go
func Execute() int {
    root := NewRoot(productionDeps())
    if err := root.Execute(); err != nil {
        fmt.Fprintf(os.Stderr, "Error: %s\n", formatError(err))
        return exitCode(err)
    }
    return 0
}
```

**JSON mode:** when `--json` is active and a command fails, the error is
written to stderr as a JSON object so machine consumers can parse it:

```json
{"error": "container \"havn-user-api\" not found"}
```

Stdout receives nothing on error, preserving stream separation. The
consumer detects failure via exit code and reads the error from stderr.

**Error translation:** `formatError` maps domain error types to
user-facing messages per the error contracts in
[havn-overview.md](havn-overview.md):

```go
func formatError(err error) string {
    var notFound *container.NotFoundError
    if errors.As(err, &notFound) {
        return fmt.Sprintf("container %q not found", notFound.Name)
    }
    // ... other domain error types
    return err.Error()
}
```

### Exit codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Error (default for all commands) |

The default exit code for any error is 1. Commands that need specific
exit codes (e.g., `havn doctor`: 0 = pass, 1 = warnings, 2 = errors)
carry them via a typed error:

```go
type ExitError struct {
    Code int
    Err  error
}
```

`Execute()` checks for `ExitError` and uses its code; all other errors
produce exit code 1. This keeps the general case simple and gives
individual commands an escape hatch when their spec requires it.

### Cobra's SilenceErrors and SilenceUsage

```go
rootCmd.SilenceErrors = true  // we handle error printing ourselves
rootCmd.SilenceUsage = true   // don't print usage on every error
```

Cobra's default behavior prints usage on any error, which is noisy for
domain errors. We silence both and handle formatting explicitly. For
actual usage errors (wrong flags, missing required args), Cobra still
returns a usage-specific error that we can detect and handle differently
if needed.

## 7. Not-Implemented Stub Pattern

During skeleton build-out, commands that don't have domain logic yet use
a consistent stub:

```go
var errNotImplemented = errors.New("not implemented")

func runList(cmd *cobra.Command, args []string) error {
    return fmt.Errorf("havn list: %w", errNotImplemented)
}
```

This makes the command callable and testable (verify it returns
`errNotImplemented`), wires it into the help system, and makes it obvious
what's left to build. The sentinel error is also useful for testing -- a
CLI test can assert that a stub command returns `errNotImplemented`.

When domain logic lands, the stub is replaced with the real call. The test
is updated to verify real behavior.

## 8. Testing the CLI Layer

_Ref: [test-standards.md](test-standards.md)_

### What to test

The CLI layer is a boundary. Tests verify:

1. **Flag parsing** -- flags are read correctly and passed to domain code.
2. **Argument handling** -- positional args parsed, validated, defaulted.
3. **Output mode** -- correct stream (stdout vs stderr), correct format
   (JSON vs human).
4. **Error formatting** -- domain errors translated to user-facing messages.
5. **Command wiring** -- each command is reachable and registered correctly.

CLI tests do **not** test domain logic. They inject fakes for domain
dependencies and verify the CLI layer's own behavior.

### Testing approach

Commands are tested by executing the Cobra command tree programmatically.
Tests call `NewRoot` with fake dependencies and exercise the returned
command:

```go
func executeCommand(args ...string) (stdout, stderr string, err error) {
    root := cli.NewRoot(fakeDeps())
    stdoutBuf := &bytes.Buffer{}
    stderrBuf := &bytes.Buffer{}
    root.SetOut(stdoutBuf)
    root.SetErr(stderrBuf)
    root.SetArgs(args)
    err = root.Execute()
    return stdoutBuf.String(), stderrBuf.String(), err
}
```

This tests the full command path (flag parsing, routing, execution) without
running a subprocess. `fakeDeps()` returns test doubles for all domain
interfaces.

### Example test

```go
func TestListCommand_JSONOutput(t *testing.T) {
    stdout, _, err := executeCommand("list", "--json")
    require.NoError(t, err)

    var containers []map[string]any
    err = json.Unmarshal([]byte(stdout), &containers)
    assert.NoError(t, err)
}

func TestListCommand_NoArgs(t *testing.T) {
    _, _, err := executeCommand("list")
    assert.NoError(t, err)
}

func TestStopCommand_RequiresNameOrAll(t *testing.T) {
    _, _, err := executeCommand("stop")
    assert.Error(t, err)
}
```

### Subprocess tests for exit codes

For testing exit codes and signal handling, use `exec.Command` to run the
built binary. These are integration tests (build-tagged) and should be few:

```go
//go:build integration

func TestExitCode_DoctorWarnings(t *testing.T) {
    cmd := exec.Command("go", "run", "./cmd/havn", "doctor")
    err := cmd.Run()
    // assert exit code
}
```

## 9. Dependency Injection for Commands

_Ref: [code-standards.md](code-standards.md) Section 4_

Commands need access to domain services (container runtime, config loader,
etc.). These are injected through a shared dependencies struct that
`NewRoot` accepts:

```go
// internal/cli/root.go

type Deps struct {
    // Fields added as domain packages are built.
    // Starts empty during skeleton phase.
}

func NewRoot(deps Deps) *cobra.Command {
    // build command tree with deps available via closure
}
```

The two entry paths:

- **Production:** `Execute()` constructs real implementations, passes
  them to `NewRoot()`, and runs the returned command.
- **Tests:** call `NewRoot(fakeDeps)` directly, bypassing `Execute()`
  and controlling all dependencies.

```go
func Execute() int {
    deps := Deps{
        // wire real implementations here as they land
    }
    root := NewRoot(deps)
    if err := root.Execute(); err != nil {
        // error formatting (see Section 6)
        return exitCode(err)
    }
    return 0
}
```

During skeleton phase, `Deps` starts empty and grows as domain packages
land. Commands that don't yet have domain dependencies don't need them --
they return `errNotImplemented`.

## 10. Shell Completions

Cobra generates shell completions automatically. havn exposes this via
Cobra's built-in `completion` command:

```
havn completion bash
havn completion zsh
havn completion fish
```

No custom completion logic is needed initially. As commands mature,
custom completions can be added (e.g., completing container names for
`havn stop`) by implementing Cobra's `ValidArgsFunction`.

## 11. Version

havn reports its version via `havn --version`. The version is set at build
time using `-ldflags`:

```go
var version = "dev"

root.Version = version
```

```makefile
build:
	go build -ldflags "-X github.com/jorgengundersen/havn/internal/cli.version=$(VERSION)" -o bin/havn ./cmd/havn
```

During development, the version is `dev`. Release builds inject the git
tag or commit hash.
