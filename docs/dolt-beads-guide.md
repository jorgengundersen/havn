# Dolt and beads guide

This is a derivative guide to how `havn` and `bd` work together in shared-Dolt
mode.

For the normative shared-Dolt contract, see `specs/shared-dolt-server.md`.

## Status

Shared-Dolt support is `Partial`: the command surface exists, but some
infrastructure lifecycle behavior is still being tightened against the spec.

Migration correctness and policy semantics are owned by beads/Dolt workflows,
not by `havn`.

## What this setup is

- `havn` runs one shared Dolt SQL server container: `havn-dolt`
- project containers connect to it over the configured Docker network
- each project gets its own database on that server
- `bd` is the user-facing interface for issue data; `havn` handles server and
  database lifecycle

## Shared-server model

When Dolt is enabled for a project, the intended flow is:

1. ensure `havn-dolt` exists and is running
2. verify the container is managed by `havn`
3. wait for readiness
4. ensure the project database exists
5. start or attach to the project container with shared-server beads env vars

The shared server lifecycle is independent from project containers:

- stopping a project container does not stop `havn-dolt`
- use `havn dolt start` and `havn dolt stop` for server lifecycle

## Project configuration

Enable Dolt in project config:

```toml
[dolt]
enabled = true
database = "myproject"
```

- `database` defaults to the project directory name when omitted
- image and port defaults can be set in global config

## beads integration

When a project starts with Dolt enabled, `havn` injects the shared-server env
vars beads expects:

- `BEADS_DOLT_SHARED_SERVER=1`
- `BEADS_DOLT_SERVER_HOST=havn-dolt`
- `BEADS_DOLT_SERVER_PORT=3308` or the configured Dolt port
- `BEADS_DOLT_SERVER_USER=root`
- `BEADS_DOLT_SERVER_DATABASE=<project database>`
- `BEADS_DOLT_AUTO_START=0`

This keeps beads in external/shared-server mode and prevents per-project Dolt
auto-start behavior.

## Operational command reference

### Server lifecycle

```bash
havn dolt start
havn dolt stop
havn dolt status
```

`status` reports shared-server state, not project-specific state.

Current status payloads include shared-server state plus `configured_sql_port`.
`configured_sql_port` is configuration-derived intent, not a runtime-observed
listening-port fact.

When runtime-port mismatch is suspected, verify runtime state with Docker
inspection (`docker inspect`, `docker port`, or container process inspection).
That runtime verification path is intentionally external to
`havn dolt status`.

### Database operations

```bash
havn dolt databases
havn dolt drop <name> --yes
havn dolt connect
```

- `databases` lists shared-server database names
- `drop` is non-interactive: `--yes` is required
- `connect` opens an interactive `dolt sql` shell in the shared server

### Migration and portability

```bash
havn dolt import <project-path> [--force]
havn dolt export <database> [--dest <path>]
```

- `import` and `export` are compatibility command surfaces in `havn`
- migration correctness, identity policy, rollback, and reconciliation semantics
  are owned by beads/Dolt workflows
- use beads docs and workflows as the migration policy source of truth
- `--force` allows overwrite on import when the destination database already
  exists

## Import workflow

Use this when moving from a local `.beads/dolt/...` database to shared-server
mode.

1. enable Dolt in `.havn/config.toml`
2. run:

```bash
havn dolt import .
```

3. start the project normally with `havn .`
4. use `bd` as usual inside the container

`havn` infrastructure responsibility here is limited to shared-server
availability and command execution framing. Migration policy and correctness are
outside `havn` ownership.

## Export workflow

Use this when you need a project-local copy of a shared database:

```bash
havn dolt export myproject --dest .
```

Common result path:

```text
./.beads/dolt/myproject/
```

## Backup and sync options

Two layers are available:

- Docker-volume level backup for the shared Dolt volume
- beads-level remote sync with `bd dolt push` and `bd dolt pull`

For day-to-day issue data sync between machines, prefer the beads remote-sync
commands.

## Caveats

- this guide describes intended shared-Dolt behavior, but support is still
  marked `Partial`
- per-project Dolt server mode is not supported by `havn`
- default access model is network-isolated shared-server access, not host port
  publishing
- authentication and TLS are not part of the default flow today

## Current partial-support gaps

- migration ownership and policy semantics intentionally live outside `havn`;
  use beads workflows/contracts as the authority
- `havn dolt status` does not claim runtime listening-port verification; use
  Docker-native inspection when runtime-port validation is required

When this guide and the spec disagree, follow `specs/shared-dolt-server.md`.
