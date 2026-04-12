# havn configuration guide

This guide explains where `havn` reads configuration, how values are merged, and how to inspect the effective configuration used for a project.

## Configuration sources

`havn` reads configuration from four layers:

1. global config file
2. project config file
3. environment variables
4. CLI flags

Higher layers override lower layers.

## File locations

### Global config

Default path:

```text
~/.config/havn/config.toml
```

Use `--config <path>` to point `havn` at a different global config file for that command invocation.

### Project config

Path relative to a project root:

```text
<project>/.havn/config.toml
```

Project config is optional. If present, it overrides matching global values.

## Precedence rules

For regular config fields (`shell`, `image`, `resources.*`, `dolt.*`, and similar), precedence is:

```text
flag > env var > project config > global config > built-in default
```

Example mappings:

- `--shell` overrides `HAVN_SHELL` and any `shell = "..."` value in config files
- `--cpus` overrides `HAVN_CPUS` and any `[resources].cpus` value in config files
- `--image` overrides `HAVN_IMAGE` and any `image = "..."` value in config files

### Dev environment (`env`) resolution

`env` has an additional source: `.havn/flake.nix`. Resolution order is:

1. `--env`
2. `HAVN_ENV`
3. `env` in `<project>/.havn/config.toml`
4. `<project>/.havn/flake.nix` (resolved as `path:./.havn`)
5. `env` in `~/.config/havn/config.toml`

## Minimal examples

### Global defaults (`~/.config/havn/config.toml`)

```toml
env = "github:jorgengundersen/dev-environments"
shell = "default"
image = "havn-base:latest"
network = "havn-net"

[resources]
cpus = 4
memory = "8g"
memory_swap = "12g"

[dolt]
enabled = false
port = 3308
image = "dolthub/dolt-sql-server:latest"
```

### Project overrides (`<project>/.havn/config.toml`)

```toml
shell = "go"
ports = ["8080:8080"]

[resources]
cpus = 8
memory = "16g"

[dolt]
enabled = true
database = "myproject"

[environment]
MY_API_KEY = "${MY_API_KEY}"
```

## Common configuration patterns

### Reuse existing Docker volumes

If you already have compatible XDG volumes from another tool, set:

```toml
[volumes]
data = "devenv-data"
cache = "devenv-cache"
state = "devenv-state"
```

### Enable shared Dolt per project

```toml
[dolt]
enabled = true
database = "api"
```

### Configure extra service ports

```toml
ports = ["3000:3000", "8080:8080"]
```

`--port` is separate and SSH-only (`host:<container 22>` behavior).

### Pass host environment values through

```toml
[environment]
GITHUB_TOKEN = "${GITHUB_TOKEN}"
```

`havn` reads the host variable at startup. If it is unset, startup fails with a validation error.

## Inspect effective configuration

Use `havn config show` to inspect the merged runtime config for a project.

### Human-readable output

```bash
havn config show
```

Use this for quick local inspection.

### JSON output

```bash
havn config show --json
```

Use this for scripts and automation. The output includes:

- effective values (`env`, `shell`, `resources`, `dolt`, and others)
- a `source` object that explains where key values came from (`default`, `global`, `project`, `env`, or `flag`)

Example snippet:

```json
{
  "shell": "go",
  "resources": {
    "cpus": 8,
    "memory": "16g"
  },
  "source": {
    "shell": "project",
    "resources": {
      "cpus": "flag",
      "memory": "project"
    }
  }
}
```

Interpretation:

- `shell = "go"` came from project config
- `resources.cpus = 8` came from a CLI flag in this example
- `resources.memory = "16g"` came from project config

## Recommended workflow

1. set stable defaults in `~/.config/havn/config.toml`
2. keep project-specific overrides in `<project>/.havn/config.toml`
3. use env vars for machine/session-specific values
4. use flags for one-off overrides
5. confirm with `havn config show --json`
