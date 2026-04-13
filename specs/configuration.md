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

`env` has one extra discovered source: `.havn/flake.nix`.

Resolution order is:

1. `--env`
2. `HAVN_ENV`
3. `env` in `<project>/.havn/config.toml`
4. `<project>/.havn/flake.nix`, resolved as `path:./.havn`
5. `env` in global config
6. built-in default

The discovered flake source has lower precedence than an explicit `env` value in
project config, environment, or flags.

### Command-specific runtime overrides

- `havn [path]` accepts root-only runtime flags such as `--shell`, `--env`,
  `--cpus`, `--memory`, `--port`, `--no-dolt`, and `--image`.
- `havn build` may honor `--image` and `--config` because they affect build-time
  image selection.
- `havn config show` reports the effective config for the command invocation.
  When future command-specific runtime flags are accepted on `config show`, its
  output must reflect them.
- Commands that do not accept a given runtime flag do not participate in that
  override surface.

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

`--no-dolt` is a root-only runtime override that forces the effective value of
`dolt.enabled` to `false` for that startup invocation.

### Lists

These fields append rather than replace:

- `mounts.config`
- `ports`

Lower-precedence entries appear first, then higher-precedence entries.

`--port` is SSH-only. It accepts a single host port number and contributes one
effective publish entry that maps `<host>:22`. That derived SSH publish entry is
merged with any configured `ports` entries into the final Docker publish set.

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
env = "github:jorgengundersen/dev-environments"
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

## Validation

Configuration validation happens after merge and after flake resolution.

At minimum, validation must reject:

- unreadable or invalid config syntax
- invalid resource values such as negative CPUs
- invalid `--port` values or malformed `ports` entries
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
  "env": "github:jorgengundersen/dev-environments",
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

When the effective flake comes from discovered `.havn/flake.nix`, the returned
`env` value is `path:./.havn` and the provenance for `env` remains `project`
scope for user interpretation purposes because the discovered flake is a
project-local source.

## Relationship to Other Specs

- `specs/cli-framework.md` owns flag parsing mechanics, stream separation, and
  CLI error behavior.
- `specs/havn-doctor.md` reuses this spec's effective-config and project-context
  semantics when doctor resolves what to check.
- `specs/shared-dolt-server.md` owns shared-Dolt lifecycle and safety semantics,
  but relies on this spec for how Dolt config is discovered and overridden.
