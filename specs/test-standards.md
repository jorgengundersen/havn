# Test Standards

Testing conventions for havn. Testing is the agent's primary
self-verification loop — weak tests mean agents can't self-verify.

**Format:** rule, rationale, example. For the _why_, see
[architecture-principles.md](architecture-principles.md) Section 2.

**Workflow:** red/green TDD. The cycle is:

1. **Red** — write _one_ failing test for the next behavior.
2. **Green** — write the minimal implementation to make that test pass.
3. **Refactor** — clean up while all tests stay green.
4. Repeat from 1.

One test, one implementation. Do not batch — don't write multiple tests
then implement them all at once. Each cycle is a single behavior.

---

## 1. Test Organization

### File placement

Tests live next to the code they test:

```
internal/config/
  merge.go
  merge_test.go
  validate.go
  validate_test.go
```

### Package naming

**Black-box by default.** Test files use the `_test` package suffix:

```go
package config_test

import (
    "testing"

    "github.com/jorgengundersen/havn/internal/config"
)
```

This enforces testing through the public API. If you can't test a
behavior through the public API, that's a signal to reconsider the API —
not to switch to white-box testing.

**Exception:** complex algorithms with many internal edge cases may use
the same package name to access unexported functions. Document why in a
comment at the top of the test file.

### Test data

Static test fixtures go in a `testdata/` directory next to the test file.
Go tooling ignores `testdata/` directories by convention:

```
internal/config/
  merge_test.go
  testdata/
    global.toml
    project.toml
    invalid.toml
```

## 2. Test Boundaries

_Ref: [Principles 2](architecture-principles.md)_

### Unit tests

The majority. Fast, no external dependencies.

- Pure functions: known inputs, expected outputs. No doubles needed.
- Composed units: inject test doubles for dependencies. Test the unit's
  contract, not the primitives inside it (those have their own tests).

### Integration tests

Verify that boundaries work in practice — the places where test doubles
can diverge from reality.

- Wrapper tests that hit a real Docker daemon.
- Config loading from actual TOML files on disk.
- Use build tags to separate from unit tests:

```go
//go:build integration

package docker_test
```

Run separately: `go test -tags integration ./...`

When an integration suite requires Docker and the daemon is unavailable,
tests should skip with a clear message rather than fail with low-level
connection noise. CI policy decides whether Docker-backed integration tests are
required in a given environment.

For Docker wrapper integration tests, cover the wrapper contract rather than
only Docker's raw behavior. If the wrapper exposes normalized semantics (for
example consistent prefix filtering across containers, networks, and volumes),
integration tests should verify that normalized contract against the live
daemon.

### System tests

Few, focused. End-to-end for critical paths only:

- `havn .` full startup sequence.
- `havn stop --all` multi-container shutdown.

Expensive — reserve for workflows where a failure in production would be
most costly.

### Choosing the right level

1. Can it be tested with pure inputs and outputs? **Unit test.**
2. Does it cross a system boundary (Docker, filesystem, network)?
   **Integration test.**
3. Does it verify a full user workflow end-to-end? **System test.**

Default to unit. Move up only when the lower level can't verify what
matters.

## 3. Test Doubles

_Ref: [Principles 4](architecture-principles.md) (dependency isolation),
[code-standards.md](code-standards.md) Section 4_

### Interfaces for injection

Domain code defines interfaces. Tests inject doubles through the same
constructors used in production:

```go
// production
svc := container.NewService(dockerClient, networkManager)

// test
svc := container.NewService(&fakeRuntime{}, &fakeNetwork{})
```

### Fakes vs mocks

- **Fakes** (preferred): lightweight implementations with real behavior.
  A fake runtime that tracks calls and returns configured responses.
  Reusable across tests.
- **Mocks** (when needed): use when you need to verify specific call
  sequences or arguments. Prefer fakes for most cases — mocks couple
  tests to implementation details.

### Mock the interface, not the dependency

Test doubles implement havn's own interfaces, not the external
dependency's API:

```go
// Good — doubles implement havn's interface
type fakeRuntime struct {
    containers []container.Info
    createErr  error
}

func (f *fakeRuntime) CreateContainer(ctx context.Context, opts container.CreateOpts) (string, error) {
    if f.createErr != nil {
        return "", f.createErr
    }
    // track the call, return a fake ID
    return "fake-id", nil
}

// Bad — doubles implement Docker's interface directly
type fakeDockerClient struct { ... }
```

### Shared test doubles

When a fake is reused across multiple test files, place it in an
internal test helper package:

```
internal/testutil/
  fakeruntime.go
  fakenetwork.go
```

This package is only imported by test files.

## 4. Test Patterns

### Table-driven tests

The standard Go pattern. Use for any function with multiple input/output
cases:

```go
func TestDeriveContainerName(t *testing.T) {
    tests := []struct {
        name    string
        parent  string
        project string
        want    name.ContainerName
    }{
        {
            name:    "standard path",
            parent:  "user",
            project: "api",
            want:    "havn-user-api",
        },
        {
            name:    "special characters sanitized",
            parent:  "user",
            project: "my.project",
            want:    "havn-user-my-project",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := name.DeriveContainerName(tt.parent, tt.project)
            assert.Equal(t, tt.want, got)
        })
    }
}
```

### Assertion library

Use `github.com/stretchr/testify`:

- `assert` — reports failure, continues test execution.
- `require` — reports failure, stops the test immediately.

Use `require` for preconditions and setup that must succeed. Use `assert`
for the actual verification:

```go
func TestStartContainer(t *testing.T) {
    ctx := context.Background()
    svc := container.NewService(&fakeRuntime{}, &fakeNetwork{})

    // precondition — if this fails, the rest is meaningless
    err := svc.EnsureNetwork(ctx)
    require.NoError(t, err)

    // actual test
    id, err := svc.Start(ctx, opts)
    assert.NoError(t, err)
    assert.Equal(t, "expected-id", id)
}
```

**Rationale:** TDD is the primary agent workflow. Testify provides
standardized failure output that agents parse reliably across every
red/green cycle.

### Test helpers

Use `t.Helper()` for any function that calls `t.Fatal`, `t.Error`, or
testify assertions — this ensures failure messages point to the caller,
not the helper:

```go
func requireValidConfig(t *testing.T, path string) config.Config {
    t.Helper()
    cfg, err := config.Load(path)
    require.NoError(t, err)
    return cfg
}
```

### Setup and teardown

Use `t.Cleanup()` for teardown — it runs even if the test panics:

```go
func TestIntegrationDockerCreate(t *testing.T) {
    client := newDockerClient(t)
    id := createTestContainer(t, client)
    t.Cleanup(func() {
        _ = client.RemoveContainer(context.Background(), id)
    })

    // test logic
}
```

For shared setup across multiple tests in a file, use `TestMain`:

```go
func TestMain(m *testing.M) {
    // setup
    code := m.Run()
    // teardown
    os.Exit(code)
}
```

## 5. Contract Testing

_Ref: [Principles 5, 2](architecture-principles.md)_

### Error contracts are testable

If a function's contract says "returns `NotFoundError` when the container
doesn't exist," test that:

```go
func TestStopContainer_NotFound(t *testing.T) {
    ctx := context.Background()
    runtime := &fakeRuntime{stopErr: &container.NotFoundError{Name: "gone"}}
    svc := container.NewService(runtime, &fakeNetwork{})

    err := svc.Stop(ctx, "gone")

    var notFound *container.NotFoundError
    assert.ErrorAs(t, err, &notFound)
    assert.Equal(t, "gone", notFound.Name)
}
```

### Test at the right level

Each composed unit is tested at its own level. Don't re-test primitives
through their consumers:

- `Merge` has its own tests for merge logic.
- `LoadAndValidate` (which calls `Merge`) tests the composition — does
  it call merge correctly and validate the result? It does not re-test
  every merge edge case.

### The refactor litmus test

After writing tests, ask: can you refactor the unit's internals without
breaking any tests? If no, the tests are coupled to implementation, not
contract. Fix the tests.

## 6. Naming

### Test functions

`Test<Unit>_<Scenario>`:

```go
func TestMergeConfig_ProjectOverridesGlobal(t *testing.T)
func TestDeriveContainerName_SpecialCharacters(t *testing.T)
func TestStartContainer_DaemonNotRunning(t *testing.T)
```

The name should describe the scenario, not restate the assertion.

### Subtest names

In table-driven tests, the `name` field is the subtest name. Use
lowercase, descriptive phrases:

```go
{name: "empty project config uses global defaults", ...}
{name: "port conflict returns error", ...}
```

These appear in test output as
`TestMergeConfig/empty_project_config_uses_global_defaults` — readable
at a glance.

## 7. CI Integration

_Ref: [Principles 10](architecture-principles.md)_

### Required checks before merge

1. `go test ./...` — all unit tests pass.
2. `golangci-lint run` — all linters pass.
3. `go build ./...` — clean compilation.

Integration tests (`-tags integration`) run in CI but may be gated on
Docker availability.

### Coverage

Coverage is a signal, not a target. Use it to find untested code paths,
not to chase a number. Don't skip testing a complex function because
overall coverage is "high enough." Don't write meaningless tests to bump
a metric.

### Flakiness

A flaky test is a bug. If a test fails intermittently:

1. Skip it immediately with `t.Skip("flaky: <issue link>")`.
2. File an issue.
3. Fix or delete it — a flaky test is worse than no test.

No test should depend on timing, external services (in unit tests), or
execution order.
