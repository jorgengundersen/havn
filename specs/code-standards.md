# Code Standards

Go-specific conventions for havn. Implements the _how_ for
[architecture-principles.md](architecture-principles.md).

**Format:** rule, rationale, example. For the _why_ behind each area, see
the referenced architecture-principles section.

**Go version:** latest stable.

---

## 1. Package Structure

_Ref: [Principles 1, 7, 11](architecture-principles.md)_

### Rules

- **`cmd/havn/` for the entry point.** Wiring only — parse input, construct
  dependencies, call composed units.
- **`internal/` for everything else.** havn is a CLI tool, not a library.
  Nothing is exported outside `cmd/`.
- **Domain-first package names.** `container`, `config`, `mount` — not
  `utils`, `helpers`, `common`.
- **One domain concern per package.** If a package needs "and" in its
  description, split it.
- **No `pkg/` directory.** Everything is internal.
- **Packages emerge from implementation.** Don't pre-plan the package tree.
  Create a package when a domain concern first needs one. The layout is
  a consequence of the code, not a blueprint for it.

### Imports

Group and order with `gci`:

```go
import (
    // stdlib
    "context"
    "fmt"

    // external dependencies
    "github.com/docker/docker/client"
    "github.com/stretchr/testify/assert"

    // internal packages
    "github.com/jorgengundersen/havn/internal/config"
    "github.com/jorgengundersen/havn/internal/container"
)
```

Blank line between each group. `gci` enforces this automatically.

## 2. Error Handling

_Ref: [Principles 5, 4](architecture-principles.md)_

### Custom errors

Define domain error types when callers need to distinguish failure modes:

```go
// internal/container/errors.go

type NotFoundError struct {
    Name string
}

func (e *NotFoundError) Error() string {
    return fmt.Sprintf("container %q not found", e.Name)
}
```

Use sentinel errors for simple cases where no context varies:

```go
var ErrDaemonNotRunning = errors.New("docker daemon is not running")
```

### Wrapping

Wrap with `fmt.Errorf` and `%w` to build error chains:

```go
func (s *Service) Start(ctx context.Context, cfg Config) error {
    id, err := s.runtime.CreateContainer(ctx, cfg.CreateOpts())
    if err != nil {
        return fmt.Errorf("start container %q: %w", cfg.Name, err)
    }
    return s.runtime.StartContainer(ctx, id)
}
```

Each layer adds its context. The chain reads like a call path:
`start environment: create container "havn-user-api": connection refused`

### Wrapper boundary translation

Dependency wrappers translate external errors into havn errors. External
error types never leak into domain code:

Wrappers also own semantic normalization where havn exposes a tighter contract
than the raw dependency. For example, if havn exposes a `NamePrefix` filter,
the wrapper should enforce true prefix semantics consistently even if the
underlying Docker daemon filter is broader or inconsistent across resource
types.

```go
// internal/docker/container.go (wrapper)

func (w *Client) CreateContainer(ctx context.Context, opts CreateOpts) (string, error) {
    resp, err := w.docker.ContainerCreate(ctx, ...)
    if err != nil {
        if client.IsErrNotFound(err) {
            return "", &container.ImageNotFoundError{Image: opts.Image}
        }
        return "", fmt.Errorf("docker create: %w", err)
    }
    return resp.ID, nil
}
```

### User-facing errors

Entry points (CLI commands) format errors for the user. Internal code
never formats for display — it returns structured errors:

```go
// internal/cli/start.go

if err := startEnv(ctx, cfg); err != nil {
    var notFound *container.ImageNotFoundError
    if errors.As(err, &notFound) {
        return fmt.Errorf("base image %q not found — run 'havn build' first", notFound.Image)
    }
    return err
}
```

### Structured JSON errors (TypedError)

Domain error types that carry machine-readable context implement the
`TypedError` interface (defined in `internal/cli/errors.go`):

```go
type TypedError interface {
    ErrorType() string            // stable snake_case identifier
    ErrorDetails() map[string]any // structured fields from the error
}
```

`Output.Error` detects this interface via `errors.As` and emits richer
JSON automatically. See [cli-framework.md](cli-framework.md) Section 6
for the JSON error shape.

A domain error type implements `TypedError` alongside `error`:

```go
// internal/config/errors.go

type ParseError struct {
    File   string
    Line   int
    Detail string
}

func (e *ParseError) Error() string {
    return fmt.Sprintf("parse error at %s:%d: %s", e.File, e.Line, e.Detail)
}

func (e *ParseError) ErrorType() string {
    return "config_parse_error"
}

func (e *ParseError) ErrorDetails() map[string]any {
    return map[string]any{"file": e.File, "line": e.Line, "detail": e.Detail}
}
```

**Rules:**
- `ErrorType()` returns a stable snake_case identifier. Once published,
  it never changes.
- `ErrorDetails()` keys can grow but never shrink.
- Not every domain error needs `TypedError` — implement it when the
  error carries structured context useful for JSON consumers.

### Rules summary

- Created at failure point with sufficient context.
- Wrapped with `%w` as they propagate — each layer adds context.
- Translated at wrapper boundaries — no external error types in domain code.
- Handled at entry points (CLI layer) — internal code propagates, never logs-and-returns.
- Propagate or handle, never both.
- No `panic` for expected conditions.

## 3. Type System

_Ref: [Principles 10, 1](architecture-principles.md)_

Use distinct types for semantically different values:

```go
type ContainerName string
type ImageName string
type NetworkName string
type VolumeName string
```

This prevents accidental misuse at compile time — a function expecting
`ContainerName` won't accept an `ImageName`.

### Config structs

Config types are plain data. Merging, defaulting, and validation are pure
functions that operate on them:

```go
type Config struct {
    Env       string          `toml:"env"`
    Shell     string          `toml:"shell"`
    Image     string          `toml:"image"`
    Network   string          `toml:"network"`
    Resources ResourceConfig  `toml:"resources"`
    Volumes   VolumeConfig    `toml:"volumes"`
    Mounts    MountConfig     `toml:"mounts"`
    Dolt      DoltConfig      `toml:"dolt"`
}

func Merge(global, project Config) Config { ... }
func Validate(cfg Config) error { ... }
```

### When to use named types vs aliases

- **Named types** (like `ContainerName`) when values are semantically
  distinct and should not be interchangeable.
- **Struct types** for compound data.
- **Don't over-type.** A bare `string` is fine for values that aren't
  confused with other strings in the same context.

## 4. Dependency Isolation

_Ref: [Principles 4, 12](architecture-principles.md)_

### Interface definition

Interfaces are defined by the consumer, not the implementor:

```go
// internal/container/runner.go (consumer defines what it needs)

type Runtime interface {
    CreateContainer(ctx context.Context, opts CreateOpts) (string, error)
    StartContainer(ctx context.Context, id string) error
    StopContainer(ctx context.Context, id string) error
}
```

### Compile-time assertions

Real implementations assert interface satisfaction at compile time:

```go
// internal/docker/container.go

var _ container.Runtime = (*Client)(nil)
```

This catches drift when a wrapper's method signature changes. Skip these
for test doubles — the test suite itself verifies those.

The assertion belongs on the actual concrete implementor of the consumer-defined
interface. In some cases that is a shared wrapper type like `docker.Client`; in
others it may be a thin adapter dedicated to one consumer boundary. The rule is
about asserting the real production implementor, not forcing every assertion
onto a single catch-all type.

### Wrapper structure

```go
// internal/docker/container.go

type Client struct {
    docker *client.Client  // Docker SDK client — never exposed
}

func NewClient(docker *client.Client) *Client {
    return &Client{docker: docker}
}

// CreateContainer translates havn types to Docker API calls
// and Docker responses back to havn types.
func (c *Client) CreateContainer(ctx context.Context, opts container.CreateOpts) (string, error) {
    // translate opts -> Docker config
    // call Docker API
    // translate response -> havn types
    // translate errors -> havn errors
}
```

### Constructor injection

Composed units receive dependencies through constructors:

```go
type Service struct {
    runtime container.Runtime
    network network.Manager
}

func NewService(runtime container.Runtime, network network.Manager) *Service {
    return &Service{runtime: runtime, network: network}
}
```

The CLI layer (`cmd/havn/`) wires real implementations. Tests inject
doubles.

## 5. Logging and Output

_Ref: [Principles 9](architecture-principles.md)_

### Library

Use `log/slog` from the standard library.

### Handler setup

Configure the slog handler at program start in `cmd/havn/`:

```go
func setupLogger(verbose, jsonOutput bool) *slog.Logger {
    var handler slog.Handler
    opts := &slog.HandlerOptions{
        Level: slog.LevelInfo,
    }
    if verbose {
        opts.Level = slog.LevelDebug
    }
    if jsonOutput {
        handler = slog.NewJSONHandler(os.Stderr, opts)
    } else {
        handler = slog.NewTextHandler(os.Stderr, opts)
    }
    return slog.New(handler)
}
```

Pass the logger via dependency injection (constructor parameter), not
`slog.Default()` globals.

Adopt logger DI intentionally, starting at long-lived external-system
boundaries where structured logs add diagnostic value (for example Docker or
shared-service wrappers). Do not plumb loggers through every package before
there is a clear need.

### First-pass logging boundary policy

The first logger DI rollout is intentionally narrow:

- **Wrapper boundary (now):** `internal/docker` is the first package that
  receives constructor-injected loggers. This package is the runtime boundary
  for long-lived calls to the Docker daemon, where structured diagnostics have
  the highest value.
- **Domain boundary (not yet):** domain packages such as `internal/container`,
  `internal/volume`, and `internal/doctor` do not receive loggers in the first
  pass. They continue returning typed errors and explicit results to the CLI.
- **CLI boundary:** user-facing status output remains explicit status writes via
  the output helper. Logger events are for diagnostics and must not replace
  those status messages.

Expand logger DI beyond this first pass only when a concrete domain workflow
shows repeated diagnostics value that cannot be captured cleanly at the wrapper
boundary.

### Stream separation

- **stderr**: all log output, status messages, progress.
- **stdout**: data output only (`--json` results, `havn list` output).

This ensures `havn list --json | jq` is never polluted by status messages.

User-facing status output and structured logs are separate channels even when
both write to stderr. Status messages explain what havn is doing for the user;
logs support diagnostics. Do not replace explicit status reporting with logger
calls, and do not duplicate the same event in both channels without a clear
reason.

### Log levels

| Level | Use |
|-------|-----|
| `Debug` | Underlying commands, timing, internal state. Shown with `--verbose`. |
| `Info` | User-visible actions: "Creating network...", "Starting container...". |
| `Warn` | Degraded state that isn't an error yet. |
| `Error` | Failures. Prefer returning errors over logging them — log only at the handling boundary. |

### Rules

- Never log and return the same error.
- Log messages use lowercase, no trailing punctuation.
- Include structured fields for machine-parseable context:
  `slog.String("container", name)`, not interpolated strings.

### Standard attributes

Use these attribute names project-wide. Consistent keys let log queries
work across packages without per-package discovery.

| Attribute | Type | Description | Typical layer |
|-----------|------|-------------|---------------|
| `component` | `slog.String` | Package or subsystem name (e.g. `"docker"`, `"container"`, `"dolt"`) | Set once per logger, usually at construction |
| `operation` | `slog.String` | Verb describing the action (e.g. `"start"`, `"stop"`, `"create"`, `"inspect"`) | Set per log call |
| `container_name` | `slog.String` | havn-managed container name | Domain / wrapper |
| `volume_name` | `slog.String` | havn-managed volume name | Domain / wrapper |
| `network_name` | `slog.String` | havn-managed network name | Domain / wrapper |
| `image` | `slog.String` | Container image reference | Domain / wrapper |
| `error` | `slog.Any` | The error value — use `slog.Any("error", err)` | Handling boundary only |

**Rules:**

- New attributes may be added, but existing names and semantics must not
  change once adopted.
- Prefer these standard names over ad-hoc alternatives (`container`
  instead of `ctr`, `name`, or `containerName`).
- `component` is typically set via `logger.With(slog.String("component",
  "docker"))` at construction time so every subsequent call inherits it.
- `error` appears only at the handling boundary — never at a site that
  also returns the error (see Rules above).

## 6. Static Analysis and Formatting

_Ref: [Principles 10](architecture-principles.md)_

### Non-negotiable

- `gofmt` — standard formatting, enforced on save and in CI.
- `goimports` / `gci` — import grouping and ordering.

### Linter configuration

Use `golangci-lint` with a curated set focused on correctness and
consistency:

**Correctness:**
- `govet` — catches real bugs (shadow, printf mismatches).
- `errcheck` — unchecked errors violate error contracts.
- `staticcheck` — the best single Go linter.
- `gosimple` — simplification suggestions (staticcheck suite).
- `unused` — dead code is a confusing signal for agents.

**Consistency:**
- `gci` — import group ordering.
- `revive` — configurable successor to `golint`.

**Add when justified:**
- `gocritic` — opinionated but catches real issues.
- `exhaustive` — exhaustive switch on enums (add when havn uses enums).

New linters are added when a recurring issue justifies encoding as a
check. Not preemptively.

### CI

All linters run in CI. A lint failure blocks merge. No `//nolint` without
a comment explaining why.

## 7. Go Idioms

_Ref: [Principles 1, 6](architecture-principles.md)_

### Context

`ctx context.Context` is the first parameter for any function that:
- Calls an external system (Docker, filesystem I/O, network)
- Composes units that do the above
- Could be cancelled or timed out

Thread context through the domain layer — a `havn stop --all` that
takes too long should be cancellable.

Pure functions (config merging, name derivation, validation) do not
take context.

### Function signatures

```go
// Good — context first, options struct for 3+ config params
func CreateContainer(ctx context.Context, opts CreateOpts) (string, error)

// Good — few params, no options struct needed
func DeriveContainerName(parent, project string) ContainerName

// Avoid — too many positional params
func CreateContainer(ctx context.Context, name, image, network string, cpus int, ...) error
```

Use options structs when a function takes 3+ configuration parameters.
Don't use functional options (`WithX()` pattern) unless there's a real
need for optional, extensible configuration — a struct is simpler and
more explicit.

### Naming

- **Exported**: `PascalCase`. Named by what it does in the domain:
  `DeriveContainerName`, `MergeConfig`.
- **Unexported**: `camelCase`.
- **Receivers**: short, consistent within a type. `c` for `Client`,
  `s` for `Service`. Not `self` or `this`.
- **Interfaces**: describe behavior. `Runtime`, `Manager`. Not `IRuntime`
  or `RuntimeInterface`.
- **Test files**: `foo_test.go` next to `foo.go`.

### Concurrency

When composing independent units (e.g., starting multiple containers),
use `errgroup` for structured concurrency:

```go
g, ctx := errgroup.WithContext(ctx)
for _, c := range containers {
    g.Go(func() error {
        return startContainer(ctx, c)
    })
}
if err := g.Wait(); err != nil {
    // errgroup returns the first error only
}
```

For best-effort multi-unit operations
([Principles 6](architecture-principles.md)), collect all results rather
than failing on the first error. Use a results channel or synchronized
slice — not `errgroup` alone, since it cancels on first error.
