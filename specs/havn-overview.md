# havn - Overview

## Overview

`havn` is a CLI tool written in Go that manages the lifecycle of development
environment containers. It provides isolated, reproducible dev environments
powered by Nix, using Docker as the runtime. The tool orchestrates container
creation, volume management, bind mounts, and optional services (Dolt) --
it does not own the environment contents (that's the dev-environments Nix flake).

**Name:** `havn` (Norwegian for "port/harbor" -- a place where containers dock)

## Architecture

```
Host machine (WSL2 / Linux)
├── havn CLI (Go binary, runs on host)
├── ~/.config/havn/config.toml (global defaults)
│
├── Project A/
│   └── .havn/
│       ├── config.toml (project overrides)
│       └── flake.nix (optional, project-specific dev environment)
│
└── Docker
    ├── havn-base:latest (minimal image with Nix + devuser)
    ├── project-a container (one per project)
    ├── project-b container (one per project)
    ├── havn-dolt container (optional, shared Dolt server)
    │
    ├── havn-nix volume (shared Nix store)
    ├── havn-data volume (XDG_DATA_HOME)
    ├── havn-cache volume (XDG_CACHE_HOME)
    ├── havn-state volume (XDG_STATE_HOME)
    ├── havn-dolt-data volume (Dolt databases)
    └── havn-dolt-config volume (Dolt server config)
```

## CLI Interface

### Primary command

```
havn [flags] [path]
```

**Default path:** `.` (current directory)

**Behavior:**
1. Resolve `path` to absolute path. Must be under `$HOME`.
2. Derive deterministic container name: `havn-<parent>-<project>`
   (e.g., `~/Repos/github.com/user/api` -> `havn-user-api`).
3. If container is running: exec into it with the activated devShell
4. If container does not exist: create and start it, then exec in.

On successful attach, `havn [path]` exits with the exit code from the shell
session it launched. If startup or attach fails, the normal CLI error handling
path applies.

This matches the previous `devenv .` behavior -- one command to start or attach.

### Subcommands

| Command | Description |
|---------|-------------|
| `havn [path]` | Start container (or attach if running) and exec into it |
| `havn list` | List running havn containers |
| `havn stop [name\|path]` | Stop a specific container |
| `havn stop --all` | Stop all havn containers |
| `havn build` | Build the base image |
| `havn volume list` | List havn volumes |
| `havn config show` | Show effective config (global + project merged) |
| `havn doctor` | Diagnose environment health ([spec](havn-doctor.md)) |
| `havn dolt start` | Start shared Dolt server container ([spec](shared-dolt-server.md)) |
| `havn dolt stop` | Stop shared Dolt server container |
| `havn dolt status` | Show Dolt server status |
| `havn dolt databases` | List all databases on the shared server |
| `havn dolt drop <name>` | Drop a project database (requires `--yes`) |
| `havn dolt connect` | Open a Dolt SQL shell for debugging |
| `havn dolt import <path>` | Import a project's local Dolt database to shared server |
| `havn dolt export <name>` | Export a database from shared server to project directory |

### Global flags

| Flag | Env Var | Description |
|------|---------|-------------|
| `--json` | | Machine-readable JSON output (see [JSON output](#json-output)) |
| `--verbose` | | Show detailed output including underlying commands and their results |
| `--shell <name>` | `HAVN_SHELL` | devShell to activate (default, go, rust, etc.) |
| `--env <flake-ref>` | `HAVN_ENV` | Nix flake ref for dev environment (see flake resolution) |
| `--cpus <n>` | `HAVN_CPUS` | CPU limit (default: 4) |
| `--memory <size>` | `HAVN_MEMORY` | Memory limit (default: 8g) |
| `--port <port>` | `HAVN_SSH_PORT` | Publish container SSH on host port `<port>` |
| `--no-dolt` | | Skip Dolt server even if project config enables it |
| `--image <name>` | `HAVN_IMAGE` | Override base image |
| `--config <path>` | | Path to config file |

All flags override config file values. Env vars override config file values.
Flags override env vars. Priority: **flag > env > project config > global config > default**.

`--port` is SSH-only. It publishes host port `<port>` to container port `22`.
If unset, SSH is available only inside the container and Docker network.

### Output modes

**Stream separation:** status and progress messages go to **stderr**,
data and results go to **stdout**. This ensures unix piping works
correctly — `havn list --json | jq '.[]'` is never polluted by status
messages.

**Normal (default):** minimal but informative. The user should always know
what havn is doing — creating a network, building an image, starting a
container. One line per action, no noise. (These go to stderr.)

```
Building base image...
Creating network havn-net...
Creating volume havn-nix...
Starting container havn-user-api...
```

**Verbose (`--verbose`):** includes underlying commands, their output, and
timing. Useful for diagnosing slow steps or unexpected failures. (Also stderr.)

```
Building base image...
  docker build -t havn-base:latest docker/
  [docker build output...]
  done (42s)
Creating network havn-net...
  docker network create havn-net
  done (0.1s)
```

### JSON output

`--json` is a global flag. Any command that displays information supports it.
Without `--json`, output is human-friendly and visually organized. With
`--json`, output is structured and stable for programmatic consumption by
AI agents and shell scripts.

Commands that only perform actions (e.g., `havn stop`, `havn build`) output
a JSON result object with `status` and `message` fields when `--json` is set.

#### `havn list --json`

```json
[
  {
    "name": "havn-user-api",
    "path": "/home/devuser/Repos/github.com/user/api",
    "image": "havn-base:latest",
    "status": "running",
    "shell": "go",
    "cpus": 4,
    "memory": "8g",
    "dolt": true
  }
]
```

Empty list → `[]`.

Only running havn-managed project containers are listed. Stopped containers are
omitted.

#### `havn volume list --json`

```json
[
  {
    "name": "havn-nix",
    "mount": "/nix",
    "exists": true
  },
  {
    "name": "havn-data",
    "mount": "/home/devuser/.local/share",
    "exists": true
  }
]
```

#### `havn config show --json`

Outputs the fully merged effective config (global + project + env + flags)
as a JSON object. The structure mirrors the TOML config schema:

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
    "config": [".gitconfig:ro", ".config/git/config:ro"],
    "ssh": {
      "forward_agent": true,
      "authorized_keys": true
    }
  },
  "dolt": {
    "enabled": true,
    "database": "myproject",
    "port": 3308,
    "image": "dolthub/dolt-sql-server:latest"
  },
  "source": {
    "env": "global",
    "shell": "project",
    "image": "default",
    "network": "default",
    "resources": {
      "cpus": "project",
      "memory": "project",
      "memory_swap": "global"
    },
    "dolt": {
      "enabled": "project",
      "database": "project",
      "port": "global",
      "image": "global"
    }
  }
}
```

The `source` object is part of the stable `havn config show --json` contract.
It shows where each effective value came from (`"default"`, `"global"`,
`"project"`, `"env"`, `"flag"`). It mirrors the shape of the returned config
object for the fields havn exposes, so callers can inspect provenance without
guessing field paths.

havn must include source metadata for these fields: `env`, `shell`, `image`,
`network`, `resources.cpus`, `resources.memory`, `resources.memory_swap`,
`dolt.enabled`, `dolt.database`, `dolt.port`, and `dolt.image`.

#### `havn dolt status --json`

```json
{
  "running": true,
  "container": "havn-dolt",
  "image": "dolthub/dolt-sql-server:latest",
  "port": 3308,
  "network": "havn-net",
  "managed_by_havn": true
}
```

If the Dolt container is not running:

```json
{
  "running": false
}
```

#### `havn dolt databases --json`

```json
["api", "web", "myproject"]
```

#### `havn doctor --json`

Defined in [havn-doctor.md](havn-doctor.md).

## Configuration

### Global config: `~/.config/havn/config.toml`

```toml
# Default Nix flake reference for dev environments.
# Projects can override this with .havn/flake.nix or .havn/config.toml env field.
env = "github:jorgengundersen/dev-environments"

# Default devShell to activate
shell = "default"

# Base image
image = "havn-base:latest"

# Network
network = "havn-net"

[resources]
cpus = 4
memory = "8g"
memory_swap = "12g"

# Volume names (override to reuse existing volumes from other tools)
[volumes]
nix = "havn-nix"
data = "havn-data"       # e.g., "devenv-data" to reuse existing
cache = "havn-cache"     # e.g., "devenv-cache"
state = "havn-state"     # e.g., "devenv-state"

# Bind mounts from host (only mounted if they exist on host)
# Paths are relative to $HOME unless absolute
[mounts]
config = [
  ".gitconfig:ro",
  ".gitconfig-*:ro",         # wildcard support
  ".config/git/config:ro",
  ".config/nvim/:ro",
  ".config/starship/:ro",
  ".config/gh/:ro",
  ".claude/.credentials.json:rw",
  ".claude/settings.json:ro",
]

[mounts.ssh]
forward_agent = true         # mount SSH_AUTH_SOCK
authorized_keys = true       # mount ~/.ssh/authorized_keys

# Dolt shared server
[dolt]
enabled = false              # global default; projects opt-in
port = 3308                  # server port (internal to havn-net, not exposed to host)
image = "dolthub/dolt-sql-server:latest"
```

### Project config: `<project>/.havn/config.toml`

```toml
# Override devShell for this project
shell = "go"

# Override the Nix flake reference for this project's dev environment.
# Default: .havn/flake.nix if it exists, otherwise the global env setting.
# env = "github:user/custom-env"

# Additional host:container port mappings for project services
ports = ["8080:8080", "3000:3000"]

# Additional bind mounts specific to this project
[mounts]
config = [".config/some-tool/:ro"]

# Dolt configuration (uses shared Dolt server)
[dolt]
enabled = true
database = "myproject"       # database name on the shared server (default: directory name)

# Additional environment variables
[environment]
MY_API_KEY = "${MY_API_KEY}" # passthrough from host

# Resource overrides
[resources]
cpus = 8
memory = "16g"
```

### Dev environment flake resolution

When determining which Nix flake to use for a project's dev environment,
havn follows this priority (highest wins):

1. **`--env` flag** — `havn --env "path:./my-flake" .`
2. **`HAVN_ENV` env var**
3. **`env` in `.havn/config.toml`** — project-level override
4. **`.havn/flake.nix`** — if this file exists, havn uses `path:./.havn`
5. **`env` in `~/.config/havn/config.toml`** — global default

This keeps project-specific dev environment flakes inside `.havn/`, avoiding
conflicts with a project's own `flake.nix` (which may define build outputs,
packages, or other non-dev concerns). The havn repo's root `flake.nix` is
reserved for building and installing havn itself.

### Port exposure model

- `--port <port>` publishes host port `<port>` to container port `22` for SSH.
- `ports = ["HOST:CONTAINER", ...]` publishes additional project service ports.
- `--port` and `ports` are separate surfaces. `--port` is only for SSH; it does
  not accept a general Docker mapping string.
- If a requested host port is already in use, container startup fails with the
  underlying Docker port-allocation error.

### Project environment resolution

The `[environment]` table defines extra environment variables for the project
container.

- Environment entries are merged by key across config files:
  1. Global `[environment]` entries are applied first.
  2. Project `[environment]` entries are applied second.
  3. For duplicate keys, the project value wins.
- Value resolution happens at container startup, after config merge:
  - Literal values are passed through unchanged.
  - `${VAR}` means "read `VAR` from the host environment and copy that value".
  - `${VAR}` passthrough is exact-token only; havn does not perform partial
    string interpolation inside larger strings.
- If a referenced host variable is unset, startup fails with a validation
  error. havn does not silently substitute an empty string.
- Final runtime env assembly is deterministic:
  1. havn mount/runtime setup env (for example `SSH_AUTH_SOCK`)
  2. user config `[environment]` entries (after `${VAR}` resolution)
  3. havn-managed integration env (for example `BEADS_DOLT_*`)
- User-defined `[environment]` entries must not override havn-managed runtime
  variables (including `SSH_AUTH_SOCK` and `BEADS_DOLT_*`). If a user sets one
  of these reserved names, startup fails with a validation error.

### Resource override surface

`memory_swap` is a config-only setting. It can be set in config files, but havn
does not expose a `--memory-swap` flag or `HAVN_MEMORY_SWAP` env var until a
concrete user need emerges.

## Container Lifecycle

### Startup sequence

```
havn .
  │
  ├─ 1. Load config (global + project, merge with flag/env overrides)
  ├─ 2. Resolve project path, derive container name
  ├─ 3. Check if container is running
  │     ├─ YES: docker exec -it --workdir <path> <name> nix develop <ref>#<shell> -c bash
  │     └─ NO: continue
  ├─ 4. Ensure base image exists (build if missing)
  ├─ 5. Ensure docker network exists (create if missing)
  ├─ 6. Ensure named volumes exist (create if missing)
  ├─ 7. If dolt.enabled:
  │     ├─ Ensure shared Dolt container is running (start if needed)
  │     └─ Ensure project database exists (CREATE DATABASE IF NOT EXISTS)
  ├─ 8. docker run -d --rm <all mounts and config> <image> sleep infinity
  ├─ 9. Run container init (start sshd)
  └─ 10. docker exec -it --workdir <path> <name> nix develop <ref>#<shell> -c bash
```

Steps 4-7 are self-healing -- havn creates what's missing as part of
normal startup. When it does, it tells the user what it's doing (e.g.,
"Network havn-net not found, creating..."). Errors only surface when
havn's own recovery fails.

**On failure:** abort the startup and tell the user what failed and why.
Do not roll back successfully completed steps. Infrastructure created
during startup (network, volumes, base image, Dolt server) is shared
across projects and useful beyond this single attempt — tearing it down
would just force the next retry to recreate it.

**Error contracts:**

| Step | Error condition | User-facing message |
|------|----------------|---------------------|
| 1 | Config file has syntax error | Config parse error at `<file>:<line>`: `<detail>` |
| 1 | Invalid config values (e.g., negative cpus) | Invalid config: `<field>`: `<reason>` |
| 2 | Path is not under `$HOME` | Project path must be under your home directory |
| 2 | Path does not exist | Directory not found: `<path>` |
| 3 | Docker daemon not running | Docker is not running. Start Docker and try again |
| 4 | Image build fails | Base image build failed: `<docker build error>`. Run `havn build` for full output |
| 5 | Network creation fails | Failed to create network '`<name>`': `<docker error>` |
| 6 | Volume creation fails | Failed to create volume '`<name>`': `<docker error>` |
| 7 | Dolt container fails to start | Failed to start Dolt server: `<error>`. Run `havn doctor` to diagnose |
| 7 | Dolt health check times out | Dolt server started but not responding. Check `docker logs havn-dolt` |
| 7 | Database creation fails | Failed to create database '`<name>`': `<sql error>` |
| 8 | Container creation fails | Failed to create container '`<name>`': `<docker error>` |
| 9 | Container init fails | Container started but init failed: `<error>`. Run `havn doctor` to diagnose |
| 10 | Exec into container fails | Failed to exec into container '`<name>`': `<docker error>` |

### Shutdown

**Single container:** `havn stop <name|path>` stops the specified container.
Containers are auto-removed (`--rm`).

**All containers:** `havn stop --all` stops all project containers matching
the `managed-by=havn` label. Best-effort -- a failure to stop one container
does not abort the rest. Reports the full outcome:

```
Stopped havn-user-api
Stopped havn-org-web
Failed to stop havn-user-dashboard: <error>

2 stopped, 1 failed
```

`havn stop --all` does not stop the shared Dolt container. Dolt is
infrastructure, not a project container. Use `havn dolt stop` explicitly.

**Dolt lifecycle is independent.** The shared Dolt container is managed
exclusively through `havn dolt start` and `havn dolt stop`. Stopping
project containers has no effect on the Dolt server.

## Base Image

`havn build` creates a minimal base image: Ubuntu 24.04 with Nix installed,
a `devuser` account, and sshd. No programming languages, editors, or tools
beyond the bare minimum -- all tooling comes from `nix develop` at runtime.
Concrete Dockerfile, build-context, and runtime details live in
[base-image.md](base-image.md).

**Why Ubuntu, not `nixos/nix`:** The official Nix image is Alpine-based
(musl libc). Many Nix packages assume glibc, causing subtle runtime issues.
Ubuntu gives glibc compatibility and multi-user Nix with the daemon, which
is more robust for the shared `/nix` volume scenario.

### User ID mapping

The `devuser` account inside the container must have the same UID and GID as
the host user. Bind-mounted files (project directory, config files) are owned
by the host user's UID -- if the container user has a different UID, file
operations fail or produce root-owned files on the host.

`havn build` (and startup auto-build) detect the host user's UID/GID and pass
them into the shared base-image build contract so the container's `devuser`
matches. This avoids permission issues without requiring the user to configure
anything.

## Volume and Mount Strategy

### Named volumes (shared across all project containers)

| Volume | Container Mount | Purpose |
|--------|----------------|---------|
| `havn-nix` | `/nix` | Shared Nix store (see Nix Store section) |
| `havn-data` | `/home/devuser/.local/share` | XDG_DATA_HOME |
| `havn-cache` | `/home/devuser/.cache` | XDG_CACHE_HOME |
| `havn-state` | `/home/devuser/.local/state` | XDG_STATE_HOME |
| `havn-dolt-data` | `/var/lib/dolt` (on havn-dolt container) | Dolt databases (shared mode only) |
| `havn-dolt-config` | `/etc/dolt/servercfg.d` (on havn-dolt container) | Dolt server config (shared mode only) |

The first four volume names are configurable in the global config (see `volumes` section).
This allows reuse of existing volumes from other tools (e.g., `devenv-data`)
without copying data.

### Bind mounts

**Project directory (read-write):**
```
$HOME/Repos/github.com/user/project -> /home/devuser/Repos/github.com/user/project
```

**Config files (from host, typically read-only):**
Mounted conditionally -- only if the host file/directory exists. See the
`mounts.config` section in the global config for the default list.

**SSH agent forwarding:**
```
$SSH_AUTH_SOCK -> /ssh-agent (ro)
SSH_AUTH_SOCK=/ssh-agent (env var set in container)
```

### Wildcard mount support

Patterns like `.gitconfig-*` expand to all matching files on the host.
This supports git's `includeIf` with directory-specific gitconfig files.

## Nix Store Strategy

### Recommendation: shared `/nix` volume

The configured Nix volume (default: `havn-nix`) is shared across all project
containers.

**Pros:**
- Nix store deduplicates aggressively -- shared packages are stored once
- After the first project runs `nix develop`, subsequent projects with
  overlapping dependencies start in seconds, not minutes
- Disk savings are significant (a typical Nix store is 5-20 GB)

**Cons:**
- `nix-collect-garbage` in one container affects all containers. A store
  path used by project A could be GC'd if project B runs garbage collection
  while A is stopped.
- A corrupted store (rare, but possible from unclean shutdown) affects all
  containers
- Concurrent `nix develop` in multiple containers may contend on the Nix
  store lock (Nix handles this with SQLite locking, so it's safe but may
  serialize builds)

**Mitigations:**
- GC is explicit in Nix (never automatic). A future `havn nix gc` command
  could enforce that no project containers are active before running GC.
- Store corruption is extremely rare with Nix's content-addressed design.
- Lock contention resolves automatically; builds queue rather than fail.

## Dolt Integration

### Strategy: Shared Dolt server

havn runs a single shared `havn-dolt` container that serves all project
databases. This is the only supported Dolt mode -- there is no per-container
option. See [shared-dolt-server.md](shared-dolt-server.md) for the full spec.

```
Project containers ──(havn-net)──> havn-dolt:3308
```

- A dedicated `havn-dolt` container runs a Dolt SQL server on the `havn-net`
  Docker network
- Data lives on the persistent `havn-dolt-data` volume
- Each project gets its own database (auto-created by `havn`)
- Projects enable Dolt with `dolt.enabled: true` and optionally set
  `dolt.database: <name>` in `.havn/config.toml`
- No Dolt binary needed inside project containers
- No per-container init script for Dolt lifecycle management

**Why not per-container?** The previous `devenv` tool ran Dolt inside each
project container. This required Dolt in the devShell, 20+ lines of stale
PID/lock cleanup in the entrypoint, and made cross-project queries impossible.
The shared server eliminates all of this complexity while reducing resource
usage. See the trade-off analysis in [shared-dolt-server.md](shared-dolt-server.md).

### Dolt auth

No authentication is needed. The Dolt server is only accessible within the
`havn-net` Docker network (not exposed to the host). The only clients are
project containers that `havn` explicitly connects to the network.

If auth becomes needed later, Dolt's native MySQL-compatible user/grant
system can be configured in the Dolt server's `config.toml`.

### Beads integration

When `havn` detects `.beads/` in a project directory and Dolt is enabled:

1. Ensure the shared Dolt container is running (start if needed)
2. Ensure the project's database exists (`CREATE DATABASE IF NOT EXISTS`)
3. Set beads env vars in the project container:
   - `BEADS_DOLT_SHARED_SERVER=1`
   - `BEADS_DOLT_SERVER_HOST=havn-dolt`
   - `BEADS_DOLT_SERVER_PORT=3308`
   - `BEADS_DOLT_SERVER_USER=root`
   - `BEADS_DOLT_SERVER_DATABASE=<database>` (from `.havn/config.toml`)
   - `BEADS_DOLT_AUTO_START=0`
4. `bd` connects to the shared server transparently

## Docker Network

`havn` creates a bridge network called `havn-net` (configurable).
All project containers and the shared Dolt container (if used) join this
network. This enables:

- Container-to-container communication by name (e.g., `havn-dolt:3308`)
- Network isolation from other Docker workloads
- No host port allocation needed for inter-container services

## Entrypoint and Init

Containers use `tini` as PID 1 (zombie reaping) with `sleep infinity` as
the main process. After container start, havn runs a post-start init
(start sshd, best-effort). Each shell session attaches via
`docker exec -it ... nix develop <ref>#<shell> -c bash`, so the user
lands directly in the activated devShell.

Concrete runtime assumptions for `tini`, `sleep`, `sudo`, and `sshd` live in
[base-image.md](base-image.md).

## Diagnostics

`havn doctor` inspects the health of the havn environment and reports
problems with actionable recommendations. Diagnostic-only — it never
creates, modifies, or deletes anything.

```
havn doctor [--json] [--all] [--verbose]
```

| Flag | Description |
|------|-------------|
| `--json` | Machine-readable JSON output |
| `--all` | Check all running havn containers (default: current directory only) |
| `--verbose` | Show detailed output per check (versions, paths, timing) |

**Exit codes:** `0` = all passed, `1` = warnings, `2` = errors.

Doctor runs in two tiers:

- **Tier 1 (host-level, always runs):** Docker daemon, base image, network,
  volumes, config validity, Dolt server health, project database existence.
- **Tier 2 (container-level, running containers only):** Nix store access,
  devShell evaluation, bind mounts, SSH agent forwarding, network
  connectivity, beads health (delegates to `bd doctor`).

Checks that depend on a failed prerequisite are skipped (e.g., if Docker
is down, all Docker-dependent checks are skipped). On a fresh install,
missing volumes and network are expected — doctor notes this rather than
alarming the user.

_Check definitions, identifiers, JSON schema, and behavior details in
[havn-doctor.md](havn-doctor.md)._

## Resolved Decisions

- **Image registry:** `havn build` is local-only for now. No registry push support.
- **Multiple projects in one container:** Not supported. One container per
  project is intentional isolation. May revisit in the future.
- **Distribution:** Nix flake in this repository (anyone can point to it).
  GitHub releases for Go binaries may be added later.
- **No home volume.** The XDG volume split (`~/.local/share`, `~/.cache`,
  `~/.local/state`) is sufficient. A home volume adds layered mount
  complexity for little gain. Shell history can be persisted by configuring
  `HISTFILE` to point into `~/.local/state/`.
- **Nix environment activation:** havn wraps the exec command —
  `docker exec ... nix develop <ref>#<shell> -c bash`. The user lands
  directly in the activated devShell. The container knows nothing about
  Nix activation; havn handles it from the outside. No extra dependencies,
  no files to manage inside the container.
