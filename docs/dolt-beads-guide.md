# Dolt and beads guide

This guide explains how `havn` and `bd` work together when projects use the shared Dolt server.

## What this setup is

- `havn` runs one shared Dolt SQL server container: `havn-dolt`.
- Project containers connect to it over the Docker network (default `havn-net`).
- Each project gets its own Dolt database on that server.
- `bd` is the user-facing interface for issue data; `havn` only handles server and database lifecycle.

At a high level:

- infrastructure is managed by `havn` (`havn dolt ...`)
- issue data operations are managed by `bd` (`bd ready`, `bd create`, `bd close`, `bd dolt push`, ...)

## Shared server model

When Dolt is enabled for a project, startup follows this flow:

1. Ensure `havn-dolt` exists and is running.
2. Verify the container is managed by `havn` (ownership label).
3. Wait for server readiness (`SELECT 1`).
4. Ensure project database exists (`CREATE DATABASE IF NOT EXISTS`).
5. Start or attach to the project container.

The Dolt server lifecycle is independent of project containers:

- stopping a project container does not stop `havn-dolt`
- use `havn dolt start` / `havn dolt stop` for shared server lifecycle

## Project configuration

Enable Dolt in project config:

```toml
[dolt]
enabled = true
database = "myproject"
```

- `database` defaults to the project directory name when omitted.
- global defaults (for image/port/enabled) can be set in `~/.config/havn/config.toml`.

## beads integration

When a project container starts with Dolt enabled, `havn` injects the shared-server environment variables beads expects:

- `BEADS_DOLT_SHARED_SERVER=1`
- `BEADS_DOLT_SERVER_HOST=havn-dolt`
- `BEADS_DOLT_SERVER_PORT=3308` (or configured Dolt port)
- `BEADS_DOLT_SERVER_USER=root`
- `BEADS_DOLT_SERVER_DATABASE=<project database>`
- `BEADS_DOLT_AUTO_START=0`

This keeps beads in external/shared-server mode and prevents per-project auto-started Dolt servers.

## Operational command reference

### Server lifecycle

```bash
havn dolt start
havn dolt stop
havn dolt status
```

- `status` supports `--json` and reports running state, image, network, and management ownership.

### Database operations

```bash
havn dolt databases
havn dolt drop <name> --yes
havn dolt connect
```

- `databases` supports `--json` and returns user databases.
- `drop` is non-interactive: `--yes` is required.
- `connect` opens an interactive `dolt sql` shell in the shared server container.

### Migration and portability

```bash
havn dolt import <project-path> [--force]
havn dolt export <database> [--dest <path>]
```

- `import` copies local project database data from `<project>/.beads/dolt/<dbname>/` into the shared server volume.
- `export` copies a shared-server database out to `<dest>/.beads/dolt/<dbname>/`.
- `--force` on import allows overwrite when the database already exists on the shared server.

## Import workflow (existing local beads database -> shared server)

Use this when moving from a local `.beads/dolt/...` database to shared-server mode.

1. Ensure project Dolt is enabled in `.havn/config.toml`.
2. Run:

```bash
havn dolt import .
```

3. Start project container normally with `havn .`.
4. Use `bd` as usual inside the container.

What import verifies:

- source database directory exists
- destination database visibility after copy
- best-effort project identity check (warning on mismatch)

## Export workflow (shared server -> project directory)

Use this when you need a project-local copy of a shared database:

```bash
havn dolt export myproject --dest .
```

Result path:

- `./.beads/dolt/myproject/`

## Backup and sync options

Two layers can be used:

- Dolt-volume level backup (Docker volume backup/restore)
- beads-level remote sync (`bd dolt push` / `bd dolt pull`)

For day-to-day issue data sync between machines, prefer beads remote sync commands.

## Current status and caveats

- This guide is implementation-first and reflects behavior currently wired in the CLI and Dolt domain package.
- Shared Dolt server mode is the supported model; per-project Dolt server mode is not part of the current `havn` flow.
- Default setup assumes network-isolated access (no host port publishing for Dolt).
- Authentication/TLS hardening is not part of the default flow today.
- `havn` handles server/database lifecycle; issue semantics, schema, and data workflows remain in beads (`bd`).

## Troubleshooting quick checks

If shared Dolt behavior looks wrong:

```bash
havn dolt status
havn dolt databases
havn doctor --verbose
```

Then validate beads connectivity from inside the project container with:

```bash
bd doctor --json
```
