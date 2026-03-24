# havn doctor - Diagnostic Subcommand

## Overview

`havn doctor` inspects the health of the havn environment and reports problems
with actionable recommendations. It checks everything havn is reliant on or
responsible for: Docker, base image, network, volumes, Nix store, Dolt server,
and (when applicable) per-container health including beads connectivity.

Doctor is diagnostic-only -- it does not attempt to fix problems. It reports
what is wrong and, where the fix is clear, tells the user what to run.

## CLI Interface

```
havn doctor [flags]
```

| Flag | Description |
|------|-------------|
| `--json` | Output results as JSON |
| `--all` | Check all running havn containers (default: current directory only) |
| `--verbose` | Show detailed output per check (versions, paths, timing, commands) |

### Exit codes

| Code | Meaning |
|------|---------|
| `0` | All checks passed |
| `1` | One or more warnings (non-blocking issues) |
| `2` | One or more errors (something is broken) |

## Check Tiers

Doctor runs in two tiers. Tier 1 (host-level) always runs. Tier 2
(container-level) runs only when a relevant container is running.

### Tier 1: Host-level checks (always run)

These checks verify that the host environment is correctly set up for havn
to operate.

#### 1.1 Docker daemon

- **Check:** `docker info` succeeds
- **Error on failure:** Docker daemon is not running or not accessible
- **Recommendation:** Start Docker, or check that the current user is in the
  `docker` group

#### 1.2 Base image

- **Check:** `docker image inspect <image>` succeeds for the configured base
  image (default: `havn-base:latest`)
- **Warning on failure:** Base image not found
- **Recommendation:** Run `havn build`

#### 1.3 Docker network

- **Check:** `docker network inspect <network>` succeeds for the configured
  network (default: `havn-net`)
- **Warning on failure:** Network does not exist
- **Recommendation:** Network is auto-created on `havn` start; this is only
  a warning if no containers have been started yet

#### 1.4 Named volumes

- **Check:** Each configured volume exists (`docker volume inspect`).
  Volume names come from config (defaults: `havn-nix`, `havn-data`,
  `havn-cache`, `havn-state`).
  - Nix volume
  - Data volume
  - Cache volume
  - State volume
  - `havn-dolt-data` (only if Dolt is enabled)
  - `havn-dolt-config` (only if Dolt is enabled)
- **Warning on missing:** Volume does not exist
- **Recommendation:** Volumes are auto-created on first `havn` start; missing
  volumes indicate either first run or manual deletion

#### 1.5 Global config

- **Check:** `~/.config/havn/config.toml` exists and parses without error
- **Warning on missing:** No global config found (using defaults)
- **Error on parse failure:** Config syntax error with line number

#### 1.6 Project config (if in a project directory with `.havn/`)

- **Check:** `.havn/config.toml` parses without error
- **Check:** Merged config (global + project) has no conflicting or invalid
  values (e.g., invalid resource limits, unknown shell names)
- **Error on parse failure:** Config syntax error with line number
- **Warning on invalid values:** Specific field and what's wrong with it

#### 1.7 Dolt server container (if Dolt is enabled)

- **Check:** `havn-dolt` container exists and is running
- **Check:** Container has the `managed-by=havn` label
- **Check:** Server is responsive (`SELECT 1` via `docker exec havn-dolt dolt sql -q "SELECT 1"`)
- **Error if not running and Dolt is enabled:** Shared Dolt server is not running
- **Recommendation:** Run `havn dolt start`
- **Error if unresponsive:** Dolt server is running but not accepting queries
- **Recommendation:** Check `docker logs havn-dolt` for errors; consider
  `havn dolt stop && havn dolt start`
- **Warning if label missing:** Container `havn-dolt` exists but was not
  created by havn
- **Recommendation:** Remove the container or rename it to avoid conflict

#### 1.8 Dolt server databases (if Dolt server is running and in a project directory)

- **Check:** The project's configured database exists on the shared server
  (`SHOW DATABASES`)
- **Warning on missing:** Database `<name>` does not exist on the shared server
- **Recommendation:** Run `havn dolt import <path>` if migrating, or
  `bd init` inside the container for a fresh start

### Tier 2: Container-level checks (running containers only)

These checks require `docker exec` into a running project container. By
default, doctor checks only the container for the current working directory.
With `--all`, it checks every running `havn-*` container.

If no relevant container is running, tier 2 is skipped with an informational
message.

#### 2.1 Nix store accessible

- **Check:** `/nix/store` is mounted and readable
  (`docker exec <container> test -d /nix/store`)
- **Error on failure:** Nix store volume is not mounted or is corrupt
- **Recommendation:** Stop the container and restart with `havn .`; if
  persistent, inspect the `havn-nix` volume

#### 2.2 Nix devShell resolves

- **Check:** The configured flake ref and shell can be evaluated
  (`docker exec <container> nix develop <ref>#<shell> --command true`)
- **Warning on failure:** Nix devShell failed to evaluate
- **Recommendation:** Check the flake ref in config; run
  `nix develop <ref>#<shell>` manually for detailed errors

#### 2.3 Bind mounts

- **Check:** Project directory is mounted and writable at the expected path
  (`docker exec <container> test -w <project_path>`)
- **Check:** Config mounts (from `mounts.config`) are present at expected
  locations (read-only mounts verified with `test -r`, read-write with `test -w`)
- **Error if project mount missing/unwritable:** Project directory not
  accessible inside container
- **Warning if config mount missing:** `<file>` not mounted (may be expected
  if the file doesn't exist on the host)

#### 2.4 SSH agent forwarding

- **Check:** `SSH_AUTH_SOCK` is set and the socket exists inside the container
  (`docker exec <container> test -S "$SSH_AUTH_SOCK"`)
- **Check:** `ssh-add -l` succeeds (agent is functional)
- **Warning on failure:** SSH agent forwarding not working
- **Recommendation:** Ensure `ssh-agent` is running on the host and
  `SSH_AUTH_SOCK` is set

#### 2.5 Dolt connectivity (if Dolt is enabled)

- **Check:** Container is connected to `havn-net`
  (`docker network inspect havn-net` from host, verify container is listed)
- **Error on failure:** Container is not on the `havn-net` network
- **Recommendation:** Stop and restart the container with `havn .`

Note: Dolt server health and database existence are already verified by
tier 1 checks (1.7, 1.8). This check only verifies the container has
network access to reach the server. No in-container MySQL client is needed.

#### 2.6 Beads health (if `.beads/` exists in the project)

- **Check:** `bd` CLI is available inside the container
  (`docker exec <container> which bd`)
- **Check:** `bd doctor --json` succeeds (delegate to beads' own diagnostics)
- **Report:** Surface any errors or warnings from `bd doctor` output. Do not
  duplicate beads' checks -- just relay the results.
- **Warning if `bd` not found:** Beads directory exists but `bd` is not
  installed
- **Recommendation:** Ensure the project's Nix devShell includes `beads`
- **Info on beads issues:** For detailed beads diagnostics, run `bd doctor`
  inside the container

The key principle: havn checks the plumbing (can the container reach Dolt?
is the database there?). Beads checks its own internals (schema version,
data integrity, sync status). Doctor bridges the two by running `bd doctor`
and surfacing its results, then referring the user to `bd doctor` for deeper
investigation.

## Output Format

### Human-readable (default)

Shows all checks with one-line status. Concise but complete — the user
sees the full picture at a glance.

```
havn doctor

Host
  [pass]  Docker daemon running
  [pass]  Base image exists
  [pass]  Network exists
  [pass]  Volumes exist
  [pass]  Global config valid
  [pass]  Project config valid
  [pass]  Dolt server running
  [pass]  Dolt database 'myproject' exists

Container: havn-user-myproject
  [pass]  Nix store mounted
  [pass]  devShell evaluates
  [pass]  Project directory writable
  [warn]  SSH agent not forwarding
         -> Ensure ssh-agent is running on host and SSH_AUTH_SOCK is set
  [pass]  Dolt network connected
  [pass]  Beads healthy

1 warning, 0 errors
```

### Verbose (`--verbose`)

Adds detail to each check: versions, paths, timing, underlying commands.

```
havn doctor --verbose

Host
  [pass]  Docker daemon running
          Docker 24.0.7, API 1.43
  [pass]  Base image exists
          havn-base:latest (built 2026-03-20)
  [pass]  Network exists
          havn-net (bridge, 3 containers connected)
  [pass]  Volumes exist
          havn-nix, havn-data, havn-cache, havn-state
  [pass]  Global config valid
          ~/.config/havn/config.toml
  [pass]  Project config valid
          .havn/config.toml (merged with global)
  [pass]  Dolt server running
          havn-dolt, dolthub/dolt-sql-server:latest, port 3308
          SELECT 1 responded in 2ms
  [pass]  Dolt database 'myproject' exists

Container: havn-user-myproject
  [pass]  Nix store mounted
          /nix/store readable, 12.4 GB
  [pass]  devShell evaluates
          github:jorgengundersen/dev-environments#go (3.2s)
  [pass]  Project directory writable
          /home/devuser/Repos/github.com/user/myproject
  [warn]  SSH agent not forwarding
          SSH_AUTH_SOCK not set inside container
         -> Ensure ssh-agent is running on host and SSH_AUTH_SOCK is set
  [pass]  Dolt network connected
          havn-user-myproject is on havn-net
  [pass]  Beads healthy
          bd doctor: 5 checks passed

1 warning, 0 errors
```

### JSON output (`--json`)

```json
{
  "status": "warn",
  "summary": {
    "passed": 10,
    "warnings": 1,
    "errors": 0
  },
  "checks": [
    {
      "tier": "host",
      "name": "docker_daemon",
      "status": "pass",
      "message": "Docker daemon running",
      "detail": "Docker 24.0.7"
    },
    {
      "tier": "container",
      "container": "havn-user-myproject",
      "name": "ssh_agent",
      "status": "warn",
      "message": "SSH agent not forwarding",
      "recommendation": "Ensure ssh-agent is running on host and SSH_AUTH_SOCK is set"
    }
  ]
}
```

Each check object:

| Field | Type | Description |
|-------|------|-------------|
| `tier` | `"host"` or `"container"` | Which tier the check belongs to |
| `container` | `string` (optional) | Container name, present only for tier 2 |
| `name` | `string` | Machine-readable check identifier |
| `status` | `"pass"`, `"warn"`, `"error"`, `"skip"` | Check result |
| `message` | `string` | Human-readable description |
| `detail` | `string` (optional) | Additional context (versions, paths, etc.) |
| `recommendation` | `string` (optional) | How to fix, present only for warn/error |

## Check Identifiers

Stable identifiers for each check, used in JSON output:

| Tier | Identifier | Check |
|------|-----------|-------|
| host | `docker_daemon` | Docker daemon accessible |
| host | `base_image` | Base image exists |
| host | `network` | Docker network exists |
| host | `volumes` | Named volumes exist |
| host | `global_config` | Global config parses |
| host | `project_config` | Project config parses and merges |
| host | `dolt_server` | Dolt container running and responsive |
| host | `dolt_database` | Project database exists on server |
| container | `nix_store` | `/nix/store` mounted and readable |
| container | `nix_devshell` | Configured devShell evaluates |
| container | `project_mount` | Project directory mounted and writable |
| container | `config_mounts` | Config bind mounts present |
| container | `ssh_agent` | SSH agent forwarding works |
| container | `dolt_connectivity` | Container connected to `havn-net` |
| container | `beads_health` | `bd doctor` passes |

## Behavior Notes

- **Check ordering:** Checks within each tier run in the order listed. If a
  check depends on a prior check (e.g., `dolt_database` depends on
  `dolt_server`), the dependent check is skipped if the prerequisite failed.
  Skipped checks report `status: "skip"` with a message indicating why.

- **Timeouts:** Each check has a timeout (default: 10 seconds). The
  `nix_devshell` check gets a longer timeout (60 seconds) since Nix evaluation
  can be slow on first run. If a check times out, it reports as an error with
  a timeout message.

- **No side effects:** Doctor never creates, modifies, or deletes anything.
  It only reads state. The sole exception is the `SELECT 1` query to Dolt,
  which is read-only.

- **Partial runs:** If Docker is down (check 1.1 fails), all subsequent checks
  that depend on Docker are skipped. Doctor still reports the config checks
  (1.5, 1.6) since those don't require Docker.

- **First-run friendliness:** Many warnings (missing volumes, missing network)
  are expected on a fresh install. Doctor should not alarm users in this state.
  Consider an informational note when multiple "auto-created on first start"
  warnings appear: "These are expected before your first `havn` run."
