# havn configuration guide

This is a derivative guide to the current configuration model.

For the normative contract, see `specs/configuration.md`.

## Status

Configuration support is `Partial`: the intended contract is defined in
`specs/configuration.md`, while some implementation details are still being
aligned to that spec.

## Configuration sources

`havn` resolves configuration from these layers:

1. built-in defaults
2. global config file
3. project config file
4. environment-variable overrides
5. command flags accepted by the current command

Higher-precedence layers override lower-precedence layers.

## File locations

### Global config

Default path:

```text
~/.config/havn/config.toml
```

Use `--config <path>` to select a different global config file for one command
invocation.

### Project config

Project-local path:

```text
<project>/.havn/config.toml
```

Project config is optional.

## Precedence rules

For normal config fields, precedence is:

```text
flag > env var > project config > global config > built-in default
```

Examples:

- `--shell` overrides `HAVN_SHELL` and any `shell` value from config files
- `--cpus` overrides `HAVN_CPUS` and any `[resources].cpus` value from config files
- `--image` overrides `HAVN_IMAGE` and any `image` value from config files

### Flake resolution

`env` has one extra discovered source from project-local flake entrypoints.
Resolution order is:

1. `--env`
2. `HAVN_ENV`
3. `env` in `<project>/.havn/config.toml`
4. discovered project flake, in this order:
   - `<project>/.havn/flake.nix` resolved as `path:./.havn`
   - `<project>/.havn/environments/default/flake.nix` resolved as `path:./.havn/environments/default`
5. `env` in global config
6. built-in default

## Minimal examples

### Global defaults

```toml
env = "path:."
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

### Project overrides

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

## Common patterns

### Reuse existing Docker volumes

```toml
[volumes]
data = "devenv-data"
cache = "devenv-cache"
state = "devenv-state"
```

### Enable shared Dolt for one project

```toml
[dolt]
enabled = true
database = "api"
```

### Configure extra service ports

```toml
ports = ["3000:3000", "8080:8080"]
```

`--port` is separate and SSH-only.

### Pass host environment values through

```toml
[environment]
GITHUB_TOKEN = "${GITHUB_TOKEN}"
```

If the host variable is unset at startup, `havn` treats that as a validation
error.

## Inspect effective configuration

Use `havn config show` to inspect the resolved configuration for the current
invocation.

### Human output

```bash
havn config show
```

### JSON output

```bash
havn config show --json
```

The JSON output includes:

- effective values such as `env`, `shell`, `resources`, `ports`, `environment`,
  and `dolt`
- a stable `source` object describing where key values came from

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

Source labels are `default`, `global`, `project`, `env`, and `flag`.

## Recommended workflow

1. put stable defaults in `~/.config/havn/config.toml`
2. keep project-specific overrides in `<project>/.havn/config.toml`
3. use environment variables for session-specific values
4. use flags for one-off overrides
5. confirm with `havn config show --json`

## Current partial-support gaps

- `havn config show` does not yet expose every provenance detail for all returned fields; the stable `source` map currently focuses on core scalar/resource and Dolt fields
- `havn config show` currently reflects startup-style effective config without command-local runtime override flags on `config show` itself

When this guide and the configuration spec disagree, follow
`specs/configuration.md`.
