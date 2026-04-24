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

- `havn dolt import/export` migration correctness semantics as a policy surface
- project-identity verification and mismatch policy during migration
- import/export correctness, rollback, and reconciliation semantics
- migration override policy and conflict resolution

These behaviors are owned by beads tooling and workflows. `havn` is responsible
for providing and operating shared Dolt infrastructure that those workflows use.

Rationale: migration correctness decisions are project/workflow policy concerns
that belong where migration state is authored and reconciled (beads/Dolt), while
`havn` owns reusable infrastructure lifecycle and command execution framing.

## Lifecycle

### Explicit lifecycle commands

- `havn dolt start`
- `havn dolt stop`
- `havn dolt status`
- `havn dolt databases`
- `havn dolt drop <name> --yes`
- `havn dolt connect`

### Shared image lifecycle contract

The shared-server image reference comes from effective config (`dolt.image`).

When `havn` needs to create the shared Dolt container and the configured image is
not present locally, `havn` attempts an automatic image pull before container
creation.

This applies to:

- explicit shared-server startup (`havn dolt start`)
- project startup paths that provision shared Dolt when Dolt is enabled

`havn` does not perform opportunistic image updates during normal startup. If
the image already exists locally, lifecycle continues without a pull.

### Pull and startup failure semantics

Image-lifecycle failures are command failures. `havn` does not report success
for `havn dolt start` or Dolt-enabled startup if image acquisition or follow-on
startup phases fail.

Failure semantics:

- image pull failure (registry auth, connectivity, rate limits, mirror policy,
  image-not-found) fails the command
- create/start/readiness failure after a successful pull also fails the command
- command errors are command-scoped and include actionable remediation guidance
  for the operator

Fresh-environment and constrained-registry expectations:

- first-run hosts without the configured image may require registry access at
  startup time
- offline or registry-constrained environments should pre-seed the configured
  image before invoking Dolt startup paths
- pre-seeding can be done with Docker-native workflows (for example pull in a
  connected environment plus `docker save`/`docker load` into the target host)

Compatibility command surfaces currently present in the CLI:

- `havn dolt import <path> [--force]`
- `havn dolt export <name> [--dest <path>]`

Command naming and output routing for these commands still follow
`specs/cli-framework.md`, but migration semantics are out of scope for this
spec and owned by beads.

### Import/export command-boundary contract

For compatibility surfaces `havn dolt import` and `havn dolt export`, `havn`
owns command execution framing only.

At the CLI boundary, these commands may report:

- command-scoped execution progress for infrastructure steps (for example import
  target resolution, shared-server availability, transfer start/completion)
- transfer-surface outcomes such as overwrite and warning signals returned by
  the command path
- explicit ownership-boundary guidance that migration semantics are owned by
  beads/Dolt workflows

They must not claim migration correctness semantics (identity policy,
reconciliation, rollback guarantees, or conflict-policy authority).

JSON success framing is command-local and stable at this boundary:

- `havn dolt import --json` returns `status`, `message`, `database`, `path`,
  `overwrote`, `warnings`, and `ownership_boundary`
- `havn dolt export --json` returns `status`, `message`, `database`, `dest`,
  and `ownership_boundary`

`ownership_boundary` is a stable discriminator indicating migration-semantics
ownership outside `havn` (current value: `beads_migration_workflow`).

Failure framing is command-scoped (`havn dolt import: ...`,
`havn dolt export: ...`) and reports command execution/infrastructure failures,
not migration correctness verdicts.

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

`configured_sql_port` in the status payload is a configuration-derived value from
effective config. It is the intended Dolt SQL port, not a runtime-observed
listening-port fact.

`havn dolt status` does not claim runtime listening-port verification. Runtime
port verification remains an external operator check when mismatch is
suspected.

Canonical shape:

```json
{
  "running": true,
  "configured_sql_port": 3308,
  "container": "havn-dolt",
  "image": "dolthub/dolt-sql-server:latest",
  "network": "havn-net",
  "managed_by_havn": true
}
```

When the server is not running:

```json
{"running": false, "configured_sql_port": 3308}
```

### Manual runtime-port verification (external)

When `configured_sql_port` appears to disagree with observed behavior, verify
runtime state outside `havn` using Docker-native inspection paths. Examples:

- inspect live process arguments inside `havn-dolt` to confirm the running
  server command port
- inspect container networking/publication details via `docker inspect` /
  `docker port` when host publishing is part of the setup

These checks are environment-specific and intentionally remain outside the
`havn dolt status` contract.

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
