# Configuration

This spec is the authoritative contract for how `havn` discovers, merges,
validates, and reports configuration.

Status: Partial

`Partial` means this document is the intended contract, but the current
implementation may not yet satisfy every detail. Derivative docs must call out
current gaps instead of presenting the whole contract as shipped.

## Ownership

This spec owns:

- global and project config discovery
- precedence across defaults, files, environment variables, and flags
- flake resolution
- merge behavior for scalar, list, map, and nested settings
- effective-config output for `havn config show`
- provenance metadata for the stable JSON contract

`specs/havn-overview.md` may summarize configuration at a high level, but it is
not authoritative for detailed config behavior.

## Config Sources

`havn` resolves configuration from these layers:

1. built-in defaults
2. global config file
3. project config file
4. environment variable overrides
5. command flags accepted by the invoked command

Higher layers override lower layers unless a field-specific rule below says
otherwise.

## Discovery

### Global config

Default path:

```text
~/.config/havn/config.toml
```

`--config <path>` changes the global config file path for the current command.
It does not change project config discovery.

Missing global config is valid. `havn` falls back to defaults.

### Project config

Project config lives at:

```text
<project>/.havn/config.toml
```

Project config is optional. Missing project config is valid.

Commands that operate on a project resolve `<project>` from their explicit path
argument when present; otherwise they use the current working directory.

## Precedence

### General precedence

For normal config fields, precedence is:

```text
flag > env > project config > global config > default
```

This applies to fields such as `shell`, `image`, `network`, `resources.*`, and
the shared-Dolt settings that expose the same override surfaces.

### Flake resolution

`env` has one extra discovered source from project-local flake entrypoints.

Resolution order is:

1. `--env`
2. `HAVN_ENV`
3. `env` in `<project>/.havn/config.toml`
4. discovered project flake, in this order:
   - `<project>/.havn/flake.nix`, resolved as `path:./.havn`
   - `<project>/.havn/environments/default/flake.nix`, resolved as `path:./.havn/environments/default`
5. `env` in global config
6. built-in default

The discovered flake source has lower precedence than an explicit `env` value in
project config, environment, or flags.

### Command-specific runtime overrides

- `havn [path]` accepts startup runtime flags `--shell`, `--env`, `--cpus`,
  `--memory`, `--port`, `--no-dolt`, and `--image`.
- `havn up [path]` accepts the same startup runtime overrides except
  `--shell`.
- `havn up [path]` also accepts startup-check modifiers `--validate` and
  `--prepare` (`--prepare` implies `--validate`).
- For `havn [path]` startup and `havn up [path]` startup, the resolved
  project path must be under the user's home directory.
- `havn build` may honor `--image` and `--config` because they affect build-time
  image selection.
- `havn config show` reports the effective config for the command invocation.
  When future command-specific runtime flags are accepted on `config show`, its
  output must reflect them.
- Commands that do not accept a given runtime flag do not participate in that
  override surface.

### Environment startup preparation capability

Environment startup preparation is command-scoped behavior, not a separate
configuration source.

The interface contract for this capability is ratified at `Status: Partial` in
`specs/environment-interface.md`; any remaining drift is runtime-alignment
follow-up, not planned contract definition.

- The optional startup preparation capability entrypoint is owned by
  `specs/environment-interface.md`.
- `havn [path]` evaluates startup preparation behavior after startup
  prerequisites are ready and before shell handoff.
- `havn up [path]` default startup does not evaluate startup preparation.
- `havn up [path] --prepare` evaluates startup preparation behavior after
  startup prerequisites are ready and before command completion.
- `havn enter [path]` remains plain-shell entry and does not run startup
  preparation.
- No new precedence layer is introduced by startup preparation. Precedence
  remains `flag > env > project > global > default`.
- Startup preparation behavior must not change `env`/`shell` resolution
  semantics or block ad-hoc `nix develop` usage.

### Startup resource application semantics

For startup commands (`havn [path]` and `havn up [path]`), resource
limits are container-scoped at creation time:

- If the resolved project container already exists (running or stopped), startup
  reuses that container and keeps its existing resource limits.
- New resource values from current config/env/flags do not mutate an existing
  project container during reuse.
- If the project container does not exist and startup creates it, resource
  limits come from the effective startup config for that invocation.
- When no custom resource overrides are supplied for creation,
  `resources.cpus=4`, `resources.memory="8g"`, and
  `resources.memory_swap="12g"` are applied.

Applied create-time resource limits must be observable immediately in container
metadata at creation time (for example via Docker inspect metadata and
havn-managed labels), not only after later lifecycle operations.

## Merge Semantics

### Scalars

Scalar fields are replaced by the highest-precedence non-empty value.

Examples:

- `shell`
- `env`
- `image`
- `network`
- `resources.cpus`
- `resources.memory`
- `resources.memory_swap`
- `dolt.port`
- `dolt.image`
- `dolt.database`

### Booleans

Boolean fields are resolved by the highest-precedence explicit setting, whether
that value is `true` or `false`.

This rule applies to:

- `mounts.ssh.forward_agent`
- `mounts.ssh.authorized_keys`
- `dolt.enabled`

`--no-dolt` is a startup runtime override for `havn [path]` and
`havn up [path]` that forces the effective value of `dolt.enabled` to `false`
for that startup invocation.

### Lists

These fields append rather than replace:

- `mounts.config`
- `ports`

Lower-precedence entries appear first, then higher-precedence entries.

`--port` is SSH-only. It accepts a single host port number and contributes one
effective publish entry that maps `<host>:22`. That derived SSH publish entry is
merged with any configured `ports` entries into the final Docker publish set.

`mounts.config` entries must resolve to paths under the user's home directory.

Startup fails if any requested host port in the final Docker publish set is not
available on the host.

### Maps

`[environment]` merges by key:

1. global values apply first
2. project values apply second
3. duplicate keys from the higher-precedence layer win

Value resolution happens after merge:

- literal strings pass through unchanged
- `${VAR}` copies the exact value of `VAR` from the host environment
- partial interpolation inside a larger string is not supported

If a referenced host variable is unset, startup fails with a validation error.

User-defined environment keys must not override havn-managed runtime variables,
including `SSH_AUTH_SOCK` and `BEADS_DOLT_*`. That is a validation error.

## Defaults

Built-in defaults:

```toml
env = "path:."
shell = "default"
image = "havn-base:latest"
network = "havn-net"

[resources]
cpus = 4
memory = "8g"
memory_swap = "12g"

[volumes]
nix = "havn-nix"
data = "havn-data"
cache = "havn-cache"
state = "havn-state"

[mounts.ssh]
forward_agent = true
authorized_keys = true

[dolt]
enabled = false
port = 3308
image = "dolthub/dolt-sql-server:latest"
```

`dolt.database` defaults to the project directory name when the effective config
enables Dolt and no explicit database name is supplied.

`memory_swap` is intentionally config-only. It has no environment-variable or
flag override surface unless a later spec revision adds one.

## Nix Registry Persistence

`havn` persists Nix flake registry aliases through the configured state volume,
not through host-global Nix config.

### Contract

- During startup and interactive entry, havn configures in-container Nix to use
  a user registry file at `/home/devuser/.local/state/nix/registry.json`.
- This file path lives under the mounted `volumes.state` location and is the
  authoritative persistence location for `nix registry` aliases inside havn.
- `nix registry add`, `nix registry remove`, and related in-container registry
  mutations persist automatically; users do not need extra bind mounts or
  manual copy steps.

### Sharing semantics

- Project containers that resolve to the same `volumes.state` value share the
  same registry alias state.
- Alias updates become visible to later sessions that mount that same state
  volume.
- If two sessions mutate the same alias, the effective result follows Nix file
  write behavior (last write wins).

### Side-effect boundaries

- havn must not mutate host-global Nix config as part of this persistence model.
- havn runtime must not rewrite `/etc/nix/nix.conf` to persist user aliases.
- Persistence is limited to paths under the mounted state volume for the target
  container.

## Validation

Configuration validation happens after merge and after flake resolution.

At minimum, validation must reject:

- unreadable or invalid config syntax
- invalid resource values such as negative CPUs
- invalid `--port` values or malformed `ports` entries
- `mounts.config` entries that resolve outside the user's home directory
- reserved `[environment]` keys that collide with havn-managed runtime env vars
- `${VAR}` passthrough entries that reference unset host variables

Config syntax failures are parse errors. Invalid merged values are validation
errors.

## Effective Config Output

`havn config show` is the authoritative inspection command for effective
configuration.

### Human output

Human-readable output may be formatted for readability. It should still reflect
the same resolved values as JSON mode.

### JSON output

`havn config show --json` writes a stable JSON object to `stdout` containing the
effective config and provenance metadata.

Expected shape:

```json
{
  "env": "path:.",
  "shell": "go",
  "image": "havn-base:latest",
  "network": "havn-net",
  "resources": {
    "cpus": 8,
    "memory": "16g",
    "memory_swap": "12g"
  },
  "volumes": {
    "nix": "havn-nix",
    "data": "havn-data",
    "cache": "havn-cache",
    "state": "havn-state"
  },
  "mounts": {
    "config": [".gitconfig:ro"],
    "ssh": {
      "forward_agent": true,
      "authorized_keys": true
    }
  },
  "environment": {
    "MY_API_KEY": "${MY_API_KEY}"
  },
  "ports": ["2222:22", "8080:8080"],
  "dolt": {
    "enabled": true,
    "database": "myproject",
    "port": 3308,
    "image": "dolthub/dolt-sql-server:latest"
  },
  "source": {
    "env": "project",
    "shell": "project",
    "image": "default",
    "network": "default",
    "resources": {
      "cpus": "flag",
      "memory": "project",
      "memory_swap": "global"
    },
    "dolt": {
      "enabled": "project",
      "database": "default",
      "port": "global",
      "image": "global"
    }
  }
}
```

The `source` object is part of the stable contract.

### Provenance metadata

The provenance map mirrors the returned config shape for the fields where `havn`
publishes source data. Source labels are:

- `default`
- `global`
- `project`
- `env`
- `flag`

`havn` must include source metadata for at least these fields:

- `env`
- `shell`
- `image`
- `network`
- `resources.cpus`
- `resources.memory`
- `resources.memory_swap`
- `dolt.enabled`
- `dolt.database`
- `dolt.port`
- `dolt.image`

When the effective flake comes from discovered project-local entrypoints, the
returned `env` value is either `path:./.havn` or
`path:./.havn/environments/default` and the provenance for `env` remains
`project` scope for user interpretation purposes.

## Relationship to Other Specs

- `specs/cli-framework.md` owns flag parsing mechanics, stream separation, and
  CLI error behavior.
- `specs/havn-doctor.md` reuses this spec's effective-config and project-context
  semantics when doctor resolves what to check.
- `specs/shared-dolt-server.md` owns shared-Dolt lifecycle and safety semantics,
  but relies on this spec for how Dolt config is discovered and overridden.
