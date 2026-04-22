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

### Startup resources: precedence and stickiness

For startup commands (`havn [path]` and `havn up [path]`), resource
values are resolved from the normal precedence chain and applied only when a
project container is created.

- if the project container already exists (running or stopped), startup reuses it
  and keeps its current limits
- new values from config/env/flags do not mutate existing container limits
- if the container is missing and startup recreates it, startup applies the
  invocation's effective resource values
- when no custom resource values are set, recreate uses defaults:
  `cpus=4`, `memory="8g"`, `memory_swap="12g"`

`memory_swap` is config-only today: there is no env var or CLI flag override for
it.

### Environment startup preparation capability

Environment startup preparation is command-scoped and capability-driven.

- startup commands (`havn [path]`, `havn up [path]`) may run an optional,
  environment-owned prepare capability when exposed by the target flake
- plain entry (`havn enter [path]`) keeps plain-shell behavior and does not run
  startup preparation
- no new configuration precedence layer is introduced by this behavior;
  precedence remains `flag > env var > project config > global config > built-in default`
- ad-hoc `nix develop` inside sessions remains supported

Normative behavior for this area lives in `specs/configuration.md` and
`specs/cli-framework.md`, with entrypoint ownership in
`specs/environment-interface.md`.

#### Reuse vs recreate at a glance

| Scenario | Result |
|---|---|
| Existing container is reused | Existing limits stay in place |
| Missing container is recreated | Effective startup limits are applied |
| Recreate with no resource overrides | Defaults `4 / 8g / 12g` are applied |

If you expect updated resource values to take effect, remove the old project
container first and then run startup again.

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

## Persistent Nix alias workflow in havn

If your `env` points at a project-local flake (for example
`path:./.havn/environments/default`), the supported shortcut is to manage
aliases from inside havn sessions.

### Why this workflow

- aliases are stored at `/home/devuser/.local/state/nix/registry.json` inside
  the container
- this path is under the mounted `volumes.state` location, so aliases persist
  across container recreation
- havn does not persist aliases by rewriting host-global Nix config

### Register and use an alias

1. start a havn session for the target project (`havn .`)
2. add the alias inside that session:

```bash
nix registry add flake:devenv "path:./.havn/environments/default"
```

3. use it normally:

```bash
nix develop devenv#codex
nix flake show devenv
```

4. exit, start a new havn session, and verify the alias is still present:

```bash
nix registry list
```

### Sharing model

- projects that resolve to the same `volumes.state` value share one alias set
- updates from one session are visible to later sessions that mount that same
  state volume
- if two sessions update the same alias, last write wins

### Migration notes

If you previously relied on host-global alias persistence for havn work:

1. start havn for the project that should own the alias behavior
2. re-add the aliases inside havn with `nix registry add`
3. verify with `nix registry list` from a fresh havn session
4. optionally remove obsolete host-global aliases to avoid ambiguity between
   host and havn-managed state

This migration keeps alias persistence scoped to havn state volumes and avoids
broad host-global side effects.

## Nix session quickstart (inside `havn enter`)

When you are inside a plain entered shell, `nix develop` without arguments may
fail if the project root has no local `flake.nix`. Use an explicit flake alias
or reference.

### Minimal flow

```bash
# one-time alias setup in a havn session
nix registry add flake:devenv "path:./.havn/environments/default"

# inspect and verify
nix registry list | rg devenv
nix flake show devenv

# enter a shell from that environment
nix develop devenv#default
# or: nix develop devenv#codex
```

### Home Manager activation (optional)

If your environment flake exposes Home Manager targets, activate one after
entering the dev shell:

```bash
nix build devenv#homeConfigurations.default.activationPackage
./result/activate
exec bash -l
```

### Quality-of-life shortcuts

- one-liner startup from a plain entered shell:

```bash
nix develop devenv#default -c bash -lc 'nix build devenv#homeConfigurations.default.activationPackage && ./result/activate && exec bash -l'
```

- shell function for repeat use:

```bash
devup() {
  nix develop devenv#default -c bash -lc 'nix build devenv#homeConfigurations.default.activationPackage && ./result/activate && exec bash -l'
}
```

- environment-owned startup prep: if your environment implements
  `apps.<system>.havn-session-prepare`, startup commands (`havn [path]`,
  `havn up [path]`) can automate session preparation. See
  `docs/environment-interface.md`.

## Current partial-support gaps

- `havn config show` does not yet expose every provenance detail for all returned fields; the stable `source` map currently focuses on core scalar/resource and Dolt fields
- `havn config show` currently reflects startup-style effective config without command-local runtime override flags on `config show` itself
- Environment startup-preparation capability contract is ratified (`Status:
  Partial` in `specs/environment-interface.md`), and full runtime alignment
  remains tracked by implementation work

When this guide and the configuration spec disagree, follow
`specs/configuration.md`.
