# Shared Dolt Server

This is the authoritative shared-Dolt contract for `havn`.

Status: Partial

## Ownership

This spec owns:

- shared Dolt lifecycle and readiness semantics
- ownership checks for the shared server container
- project database provisioning rules
- import and export behavior
- migration and project-identity safety semantics

Configuration discovery and override precedence come from
`specs/configuration.md`. CLI naming and general output rules come from
`specs/cli-framework.md`.

## Supported Model

`havn` supports one shared Dolt SQL server container, `havn-dolt`, for all
projects that opt into Dolt.

There is no supported per-project Dolt server mode in `havn`.

Overview-level consequences:

- project containers connect to the shared server over the configured Docker
  network
- each project gets a separate database on that server
- `bd` remains the primary interface for issue data inside the project
  container

## Lifecycle

### Explicit lifecycle commands

- `havn dolt start`
- `havn dolt stop`
- `havn dolt status`
- `havn dolt databases`
- `havn dolt drop <name> --yes`
- `havn dolt connect`
- `havn dolt import <path> [--force]`
- `havn dolt export <name> [--dest <path>]`

### Startup integration

When effective config enables Dolt for a project startup:

1. ensure the shared Dolt server exists and is running
2. verify the container is managed by `havn`
3. wait for readiness
4. ensure the project's database exists
5. inject the shared-server beads environment variables into the project
   container

Stopping a project container does not stop the shared Dolt server.

## Ownership And Readiness

### Ownership

`havn` only manages the shared Dolt container when it is tagged as havn-owned.

Required ownership signal:

- Docker label `managed-by=havn`

If a container named `havn-dolt` exists without the ownership label, `havn`
must treat that as a conflict rather than silently taking it over.

### Readiness

After starting or reusing the server, `havn` waits for readiness before project
database work or migration steps continue.

Canonical readiness probe: a successful read-only query such as `SELECT 1`.

Readiness failures are errors. They are not treated as successful partial start.

## Database Provisioning

When Dolt is enabled for a project startup, `havn` ensures that the project's
database exists on the shared server.

Default database name:

- `dolt.database` from effective config when set
- otherwise the project directory name

Project containers receive the beads shared-server env vars expected for the
resolved database:

- `BEADS_DOLT_SHARED_SERVER=1`
- `BEADS_DOLT_SERVER_HOST=havn-dolt`
- `BEADS_DOLT_SERVER_PORT=<effective dolt port>`
- `BEADS_DOLT_SERVER_USER=root`
- `BEADS_DOLT_SERVER_DATABASE=<resolved database>`
- `BEADS_DOLT_AUTO_START=0`

## Safety Semantics

### No hidden takeover

`havn` must never silently manage a conflicting `havn-dolt` container it did not
create.

### No rollback of shared infrastructure on startup failure

If project startup fails after shared Dolt was started or reused, `havn` does
not roll back the shared server. The shared server is infrastructure used across
projects.

### Partial-failure reporting

Shared-Dolt commands that perform multiple independent steps must report where a
failure happened. They must not report full success when a later verification
step failed.

### Project identity

When importing or otherwise connecting project data, `havn` must preserve or
check the beads project identity where the workflow exposes it.

At minimum:

- import verifies that the copied database is visible on the shared server
- when project identity can be compared between `.beads/metadata.json` and the
  database, `havn` performs that comparison
- identity mismatch is surfaced clearly; it is never silently ignored as a
  successful migration

## Import Contract

`havn dolt import <path>` migrates an existing project-local beads database into
the shared server.

Source shape:

- `<project>/.beads/dolt/<database>/`

Nominal flow:

1. resolve project path and effective config
2. resolve the database name
3. verify the source database directory exists
4. ensure the shared Dolt server is running and ready
5. check whether the destination database already exists
6. if it exists and `--force` is not set, fail without modifying the destination
7. copy the database into the shared server data location
8. verify the database is now visible on the server
9. compare project identity when available and report any mismatch

### Overwrite semantics

- without `--force`, import must fail if the destination database already exists
- with `--force`, import may overwrite the destination database, but the command
  must make the overwrite explicit in its output

### Rollback semantics

Import does not promise transactional rollback. If a copy or verification step
fails after partial destination changes, the command must report that exact
state so the user can decide whether to retry or clean up.

## Export Contract

`havn dolt export <name> [--dest <path>]` copies a database from the shared
server back into a project-local layout.

Destination shape:

- `<dest>/.beads/dolt/<name>/`

Export must fail clearly when the source database does not exist. It must not
claim success if the final copied database is missing from the destination.

## Status And Databases Output

### `havn dolt status --json`

The status payload describes shared-server state, not project state.

Canonical shape:

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

When the server is not running:

```json
{"running": false}
```

### `havn dolt databases --json`

Returns a JSON array of database names.

## Authentication And Exposure Model

Default security model:

- no host port publishing
- access limited to the configured Docker network
- no password required by default

This model is acceptable because the trust boundary is the isolated Docker
network, not the host's public interfaces.

Authentication or TLS can be added in a later spec revision. They are not part
of the default shared-Dolt contract today.

## Relationship To Derivative Docs

User-facing docs such as `docs/dolt-beads-guide.md` explain how to use the
shared server, but they should point back here for lifecycle, readiness,
ownership, import/export, and safety semantics.
