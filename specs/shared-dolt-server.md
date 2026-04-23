# Shared Dolt Server

This is the authoritative shared-Dolt contract for `havn`.

Status: Partial

## Ownership

This spec owns:

- shared Dolt lifecycle and readiness semantics
- ownership checks for the shared server container
- project database provisioning rules for startup integration
- shared-server status/databases observability contract

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

## Out Of Scope (Owned By beads)

`havn` does not define or guarantee beads data-migration semantics.

Out of scope for this spec:

- project-identity verification and mismatch policy during migration
- import/export correctness, rollback, and reconciliation semantics
- migration override policy and conflict resolution

These behaviors are owned by beads tooling and workflows. `havn` is responsible
for providing and operating shared Dolt infrastructure that those workflows use.

## Lifecycle

### Explicit lifecycle commands

- `havn dolt start`
- `havn dolt stop`
- `havn dolt status`
- `havn dolt databases`
- `havn dolt drop <name> --yes`
- `havn dolt connect`

Compatibility command surfaces currently present in the CLI:

- `havn dolt import <path> [--force]`
- `havn dolt export <name> [--dest <path>]`

Command naming and output routing for these commands still follow
`specs/cli-framework.md`, but migration semantics are out of scope for this
spec and owned by beads.

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
database work continues.

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

## Status And Databases Output

### `havn dolt status --json`

The status payload describes shared-server state, not project state.

`havn dolt status` currently does not report a runtime port. The payload is
limited to fields that the runtime adapter can report faithfully on supported
paths.

Canonical shape:

```json
{
  "running": true,
  "container": "havn-dolt",
  "image": "dolthub/dolt-sql-server:latest",
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
ownership, startup provisioning, and status/databases observability semantics.
