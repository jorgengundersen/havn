# Shared Dolt Server for havn

## Overview

This spec describes how `havn` manages a single shared Dolt SQL server
container that serves per-project databases for [beads](https://github.com/steveyegge/beads)
issue tracking. All project containers connect to this shared server over
the `havn-net` Docker network.

## Architecture

```
┌─────────────────────────────────────────────────────┐
│  havn-net (Docker bridge network)                   │
│                                                     │
│  ┌──────────────┐  ┌──────────────┐                 │
│  │ havn-user-api│  │ havn-org-web │  ...            │
│  │              │  │              │                  │
│  │ bd connects  │  │ bd connects  │                  │
│  │ to havn-dolt │  │ to havn-dolt │                  │
│  │ :3308/api    │  │ :3308/web    │                  │
│  └──────┬───────┘  └──────┬───────┘                  │
│         │                 │                          │
│         └────────┬────────┘                          │
│                  ▼                                   │
│         ┌────────────────┐                           │
│         │   havn-dolt    │                           │
│         │                │                           │
│         │ dolthub/dolt-  │                           │
│         │ sql-server     │                           │
│         │                │                           │
│         │ /var/lib/dolt/ │◄── havn-dolt-data volume  │
│         │   ├── api/     │    (persistent)           │
│         │   ├── web/     │                           │
│         │   └── .doltcfg/│                           │
│         │                │                           │
│         │ port 3308      │                           │
│         └────────────────┘                           │
└─────────────────────────────────────────────────────┘
```

## Dolt Container Setup

### Image

Use the official [`dolthub/dolt-sql-server`](https://hub.docker.com/r/dolthub/dolt-sql-server)
image. It starts `dolt sql-server` on container launch and supports
`linux/amd64` and `linux/arm64`.

Reference: [Dolt Docker Installation](https://docs.dolthub.com/introduction/installation/docker)

### Container creation

```bash
docker run -d \
  --name havn-dolt \
  --network havn-net \
  --restart unless-stopped \
  --label managed-by=havn \
  -e DOLT_ROOT_HOST='%' \
  -v havn-dolt-data:/var/lib/dolt \
  -v havn-dolt-config:/etc/dolt/servercfg.d \
  dolthub/dolt-sql-server:latest
```

Key choices:

| Setting | Value | Rationale |
|---------|-------|-----------|
| `--restart unless-stopped` | Auto-restart on crash | More resilient than `--rm`; server should outlive individual project containers |
| `DOLT_ROOT_HOST='%'` | Allow connections from any host | Required because project containers connect from different IPs on the Docker network |
| No `DOLT_ROOT_PASSWORD` | No password | Network-isolated; only reachable from `havn-net` (see Auth section) |
| No `-p` port mapping | Not exposed to host | Only accessible within `havn-net`; no host port conflicts |
| `--label managed-by=havn` | Container ownership tag | Lets havn verify it created this container before managing it |

### Server configuration

`havn` generates a `config.yaml` and mounts it at `/etc/dolt/servercfg.d/config.yaml`:

```yaml
log_level: info

listener:
  host: 0.0.0.0
  port: 3308
  read_timeout_millis: 300000
  write_timeout_millis: 300000

data_dir: /var/lib/dolt

behavior:
  autocommit: true
```

The read/write timeouts (5 minutes) prevent a runaway query in one project
from blocking the shared server indefinitely.

Reference: [Dolt Server Configuration](https://docs.dolthub.com/sql-reference/server/configuration)

The `data_dir` setting is key: Dolt auto-discovers all subdirectories
containing a `.dolt` folder and serves them as databases. New databases
created via `CREATE DATABASE` also appear as subdirectories under `data_dir`.
No per-database configuration is needed.

### Persistence

The `havn-dolt-data` Docker volume is mounted at `/var/lib/dolt/`. This
volume contains:

```
/var/lib/dolt/
├── .doltcfg/
│   └── privileges.db       # user/grant storage
├── api/                     # project "api" database
│   └── .dolt/
│       ├── config.json
│       ├── repo_state.json
│       └── noms/            # dolt storage chunks
├── web/                     # project "web" database
│   └── .dolt/
│       └── ...
```

All database state survives container restarts and rebuilds.

## Lifecycle Management

### When havn starts a project container

```
havn .
  │
  ├─ Load project config (.havn/config.toml)
  ├─ Is dolt.enabled?
  │   └─ NO: skip dolt entirely
  │   └─ YES: continue
  ├─ Is havn-dolt container running?
  │   └─ NO: start it (docker run ... or docker start havn-dolt)
  │   └─ YES: verify ownership (--label managed-by=havn)
  │         └─ Label missing: error "havn-dolt container exists but was not created by havn"
  │         └─ Label present: continue
  ├─ Health check: poll havn-dolt until ready (SELECT 1)
  ├─ Ensure project database exists:
  │   └─ CREATE DATABASE IF NOT EXISTS `<dolt.database>`
  ├─ Start project container with beads env vars:
  │   ├─ BEADS_DOLT_SERVER_HOST=havn-dolt
  │   ├─ BEADS_DOLT_SERVER_PORT=3308
  │   ├─ BEADS_DOLT_SERVER_USER=root
  │   ├─ BEADS_DOLT_SERVER_DATABASE=<database>
  │   ├─ BEADS_DOLT_AUTO_START=0
  │   └─ BEADS_DOLT_SHARED_SERVER=1
  └─ Exec into project container
```

### Dolt lifecycle

The Dolt server lifecycle is independent from project containers. Stopping
project containers has no effect on the Dolt server. Manage it explicitly:

```bash
havn dolt start       # start the shared Dolt container
havn dolt stop        # stop it
havn dolt status      # show status, port, databases
havn dolt databases   # list all databases on the server
havn dolt drop <name> # drop a project database (requires --yes)
```

## Design Principles

### Use `bd` as the interface to beads data

`havn` should treat the `bd` CLI as the primary interface for any operation
involving beads data. Rather than connecting to Dolt directly via SQL, `havn`
invokes `bd` as a subprocess with the appropriate environment variables set.

This keeps `havn` decoupled from beads internals -- beads can change its schema,
queries, or behavior without breaking `havn`. The only SQL `havn` runs directly
is `CREATE DATABASE IF NOT EXISTS` during initial setup, since this is a
server-level operation outside of beads' scope.

### `.no-sync` marker files

A project can opt out of remote sync by placing a `.no-sync` file in its
`.beads/` directory. When present, `havn` skips any remote push/pull operations
for that project's database. This is useful for scratch databases, local
experiments, or projects that don't have a Dolt remote configured.

```bash
touch .beads/.no-sync    # prevent remote sync for this project
rm .beads/.no-sync       # re-enable sync
```

## Per-Project Configuration

### .havn/config.toml

```toml
[dolt]
enabled = true
database = "myproject"     # database name on the shared server
```

The `database` field defaults to the project directory name if omitted.

### How beads sees it

When `havn` starts a project container, it sets environment variables that
beads reads to connect to the external Dolt server. From beads' perspective,
this is equivalent to running in
[external server mode](https://github.com/steveyegge/beads/blob/main/docs/DOLT.md).

| Env Var | Value | Purpose |
|---------|-------|---------|
| `BEADS_DOLT_SERVER_HOST` | `havn-dolt` | Docker DNS name of the Dolt container |
| `BEADS_DOLT_SERVER_PORT` | `3308` | Dolt server port (beads shared mode default) |
| `BEADS_DOLT_SERVER_USER` | `root` | MySQL user |
| `BEADS_DOLT_AUTO_START` | `0` | Prevent beads from starting its own Dolt server |
| `BEADS_DOLT_SHARED_SERVER` | `1` | Tell beads it's using a shared server |
| `BEADS_DOLT_SERVER_DATABASE` | `<database>` | Database name from .havn/config.toml |

Reference: beads environment variables are defined in
[`internal/configfile/configfile.go`](https://github.com/steveyegge/beads/blob/main/internal/configfile/configfile.go)
and server mode resolution in
[`internal/doltserver/servermode.go`](https://github.com/steveyegge/beads/blob/main/internal/doltserver/servermode.go).

With these env vars set, beads connects via the MySQL wire protocol:

```
root@tcp(havn-dolt:3308)/myproject?parseTime=true&timeout=5s&readTimeout=10s&writeTimeout=10s
```

### bd init in a havn container

When running `bd init` inside a project container for the first time:

```bash
bd init --prefix myproject
```

Beads detects `BEADS_DOLT_SHARED_SERVER=1`, resolves to `ServerModeExternal`,
skips auto-starting a local Dolt server, and connects to the shared one.
Since `havn` already ran `CREATE DATABASE IF NOT EXISTS`, the database exists
and `bd init` creates the schema tables (issues, dependencies, etc.) inside it.

The resulting `.beads/metadata.json`:

```json
{
  "database": "dolt",
  "backend": "dolt",
  "dolt_mode": "server",
  "dolt_database": "myproject",
  "project_id": "..."
}
```

### What happens to .beads/dolt/ in the project directory

In shared mode, the `.beads/dolt/` directory inside the project is **not used
for data storage**. The actual data lives on the `havn-dolt-data` volume.

The `.beads/` directory still exists for:
- `.beads/config.yaml` -- beads configuration
- `.beads/metadata.json` -- connection metadata and project ID
- `.beads/.beads-credential-key` -- encryption key for federation peers

The `.beads/dolt/` subdirectory can be omitted or gitignored. Beads does not
create it when connecting to an external server.

## Authentication

### Why no auth is needed

The shared Dolt server is only accessible within the `havn-net` Docker
network. It is not exposed to the host (`-p` flag is omitted). The only
clients are project containers that `havn` explicitly connects to the network.

This is equivalent to the current per-container setup where Dolt binds to
`127.0.0.1` -- the trust boundary is the same (container network vs localhost),
just at a different scope.

### If auth becomes needed

Dolt supports MySQL-compatible user management. To add authentication:

1. Set `DOLT_ROOT_PASSWORD` on the Dolt container
2. Create per-project users via an init script in `/docker-entrypoint-initdb.d/`:
   ```sql
   CREATE USER 'beads'@'%' IDENTIFIED BY 'password';
   GRANT ALL ON *.* TO 'beads'@'%';
   ```
3. Set `BEADS_DOLT_PASSWORD=password` in project containers
4. Credentials persist in Dolt's `privilege_file` (`.doltcfg/privileges.db`
   on the data volume)

Reference: [Dolt Server Configuration - Users](https://docs.dolthub.com/sql-reference/server/configuration)

For TLS, configure `tls_key` and `tls_cert` in the Dolt server config and
set `BEADS_DOLT_SERVER_TLS=1` in project containers.

This is not planned for the initial implementation.

## Trade-offs

### Advantages over per-container Dolt

| | Shared server | Per-container |
|---|---|---|
| **Resource usage** | One process for all projects | One process per active project |
| **Container simplicity** | No Dolt binary needed in project containers; no init script for Dolt | Dolt must be in devShell or image; entrypoint manages server lifecycle |
| **Startup complexity** | No stale PID/lock cleanup | 20+ lines of cleanup logic for unclean shutdowns |
| **Cross-project queries** | Possible (same server, different databases) | Not possible without network setup |
| **Data inspection** | One place to look | Scattered across project directories |

### Disadvantages

| Concern | Impact | Mitigation |
|---------|--------|------------|
| **Single point of failure** | If Dolt container crashes, all projects lose database access | `--restart unless-stopped` auto-recovers; beads operations that don't touch Dolt still work |
| **Blast radius** | A bad query in one project could theoretically affect server stability | Dolt is mature MySQL-compatible; per-database isolation is strong. If needed, add per-user grants |
| **Data not in project directory** | Can't git-track or see Dolt data alongside project files | Beads databases are typically gitignored anyway; `bd dolt push` handles backup/sync |
| **Cleanup responsibility** | Deleting a project doesn't remove its database | `havn dolt drop <name>` or `havn dolt databases` to audit; could add cleanup prompt to `havn stop` |
| **Port allocation** | Fixed port (3308) on Docker network | Not a real issue; only one Dolt server exists, no conflicts within `havn-net` |
| **Version coupling** | All projects share the same Dolt version | Acceptable for beads use case; Dolt is backwards-compatible. Pin the image tag in havn config if needed |

### Data lifecycle comparison

```
Per-container:
  project/.beads/dolt/mydb/  ── lives with project
  delete project             ── data gone
  git clone on new machine   ── no data (gitignored) or data included

Shared server:
  havn-dolt-data volume      ── lives on Docker volume
  delete project             ── data remains (orphaned until explicit drop)
  git clone on new machine   ── no data; bd init creates fresh database
  docker volume rm           ── all project databases gone
```

## Operational Commands

### Database management

```bash
# List all databases on the shared server
havn dolt databases

# Drop a specific database (requires --yes)
havn dolt drop myproject --yes

# Connect to the Dolt server with a MySQL client (for debugging)
havn dolt connect
# equivalent to: docker exec -it havn-dolt dolt sql
```

### Backup and restore

The `havn-dolt-data` volume can be backed up with standard Docker volume
backup techniques:

```bash
# Backup
docker run --rm -v havn-dolt-data:/data -v $(pwd):/backup \
  alpine tar czf /backup/havn-dolt-backup.tar.gz /data

# Restore
docker run --rm -v havn-dolt-data:/data -v $(pwd):/backup \
  alpine tar xzf /backup/havn-dolt-backup.tar.gz -C /
```

For per-project backup, use beads' built-in Dolt remote sync:

```bash
bd dolt push    # push to a Dolt remote (DoltHub, DoltLab, or self-hosted)
bd dolt pull    # pull from remote
```

## Configuration Reference

### Global: ~/.config/havn/config.toml

```toml
[dolt]
enabled = false                             # global default; projects opt in
image = "dolthub/dolt-sql-server:latest"    # Dolt Docker image
port = 3308                                 # server port inside container
```

### Per-project: .havn/config.toml

```toml
[dolt]
enabled = true           # use the shared Dolt server (default: inherit from global)
database = "myproject"   # database name (default: project directory name)
```

### Per-project: .beads/config.yaml

When using the shared server, beads config should include:

```yaml
dolt:
  auto-start: false       # havn manages the server
```

This is technically redundant when `BEADS_DOLT_SHARED_SERVER=1` is set
(beads disables auto-start automatically in external mode), but making it
explicit avoids confusion if the env var is ever missing.

## Migrating Existing Beads Databases

### The problem

Projects using the previous `devenv` tool (or standalone beads) have existing
Dolt databases at `.beads/dolt/<dbname>/` inside the project directory. When
switching to `havn` with a shared Dolt server, this data needs to move to the
`havn-dolt-data` volume -- otherwise the project starts with an empty database.

### Beads project_id validation

Beads stores a UUID (`project_id`) in two places:
1. `.beads/metadata.json` (file in the project directory)
2. `_project_id` key in the database's `metadata` table

On every connection (except `bd init`), beads verifies that these two values
match. If they don't, beads refuses to connect with:

```
PROJECT IDENTITY MISMATCH — refusing to connect

  Local project ID (metadata.json):  <local-uuid>
  Database project ID:               <db-uuid>

This means the Dolt server is serving a DIFFERENT project's database.
```

This means:
- **Copying a database from the same project works** -- the project_id in
  metadata.json matches the one stored in the database
- **Connecting a different project to someone else's database fails** --
  project_id mismatch is a hard error
- **Old projects without project_id** -- verification is skipped (both
  sides must have a project_id for the check to trigger)

Reference: project_id verification logic in
[`internal/storage/dolt/store.go`](https://github.com/steveyegge/beads/blob/main/internal/storage/dolt/store.go)
(see `verifyProjectIdentity`), introduced to fix
[cross-project data leakage (GH#2372)](https://github.com/steveyegge/beads/issues/2372).

### Migration strategies

#### Strategy 1: Direct copy (recommended)

Copy the Dolt database directory from the project into the shared server's
data volume. This preserves all data, commit history, and the project_id.

```
havn dolt import <project-path>
```

What `havn dolt import` does:

```
havn dolt import /path/to/myproject
  │
  ├─ 1. Read .havn/config.toml (or derive database name from directory name)
  ├─ 2. Locate .beads/dolt/<dbname>/ in the project directory
  │     └─ If not found: error "no existing beads database found"
  ├─ 3. Check if database already exists on shared server
  │     └─ If exists: error "database '<name>' already exists; use --force to overwrite"
  ├─ 4. Ensure havn-dolt container is running
  │     └─ Command wiring must perform the same readiness flow used by other shared-Dolt operations
  ├─ 5. Copy database directory into the havn-dolt-data volume:
  │     └─ docker cp <project>/.beads/dolt/<dbname>/ havn-dolt:/var/lib/dolt/<dbname>/
  ├─ 6. Verify Dolt picks up the database (SHOW DATABASES)
  ├─ 7. Verify project_id matches:
  │     └─ Read project_id from .beads/metadata.json
  │     └─ Query _project_id from database metadata table
  │     └─ Compare; warn if mismatch
  └─ 8. Print success message with connection details
```

After import, the old `.beads/dolt/` directory in the project can be removed
or left in place (beads won't use it when `BEADS_DOLT_SHARED_SERVER=1` is set).

#### Strategy 2: Beads backup/restore (portable, no Dolt history)

Use beads' built-in JSONL backup to export and re-import. This preserves
issues, comments, events, dependencies, and labels -- but loses Dolt commit
history (branches, diffs, etc.).

```bash
# Inside the OLD environment (devenv container or local):
bd backup > backup.jsonl

# Inside the NEW havn container (shared server already running):
bd init --prefix myproject
bd backup restore backup.jsonl
```

Reference: [`bd backup` / `bd backup restore`](https://github.com/steveyegge/beads/blob/main/docs/DOLT.md)

#### Strategy 3: Dolt remote push/pull (preserves full history)

If the project has a Dolt remote configured (DoltHub, DoltLab, or self-hosted),
push from the old setup and pull into the new one.

```bash
# Inside the OLD environment:
bd dolt remote add origin https://doltremoteapi.dolthub.com/user/myproject
bd dolt push origin main

# Inside the NEW havn container:
bd init --prefix myproject
bd dolt remote add origin https://doltremoteapi.dolthub.com/user/myproject
bd dolt pull origin main
```

This preserves full Dolt commit history, branches, and diffs. Requires a
Dolt remote (DoltHub account or self-hosted remote).

Reference: [`bd dolt push` / `bd dolt pull`](https://github.com/steveyegge/beads/blob/main/docs/DOLT.md)

### Strategy comparison

| | Direct copy | Backup/restore | Remote push/pull |
|---|---|---|---|
| **Preserves Dolt history** | Yes | No (JSONL snapshot) | Yes |
| **Preserves project_id** | Yes | No (new ID on `bd init`) | No (new ID on `bd init`) |
| **Requires remote** | No | No | Yes |
| **Speed** | Fast (file copy) | Medium (serialize/deserialize) | Slow (network transfer) |
| **Complexity** | Low | Low | Medium |
| **Works offline** | Yes | Yes | No |

Direct copy is recommended as the default migration path because it's fast,
preserves everything, and works offline.

### The export command (reverse migration)

For completeness, `havn` should also support exporting a database from the
shared server back to a project directory:

```bash
havn dolt export <dbname> [--dest <path>]
```

This copies the database directory from the `havn-dolt-data` volume to
`<project>/.beads/dolt/<dbname>/`. Useful for:
- Moving back to per-container Dolt (if ever needed)
- Creating a local backup alongside the project
- Sharing the database directory with someone who doesn't use `havn`

### Automatic migration detection

When `havn .` starts a project container and detects:
1. `dolt.enabled: true` in `.havn/config.toml`
2. An existing `.beads/dolt/<dbname>/` directory with a `.dolt/` subfolder
3. The database does NOT exist on the shared server

`havn` informs the user and continues startup normally:

```
Existing beads database found at .beads/dolt/myproject/
Not yet on shared server. Run `havn dolt import .` to migrate.
```

Startup is not blocked. The user runs the import manually when ready.
This is a one-time migration step for existing projects — no interactive
prompt needed.

## References

- [Beads - Distributed Graph Issue Tracker](https://github.com/steveyegge/beads)
- [Beads Dolt Documentation](https://github.com/steveyegge/beads/blob/main/docs/DOLT.md)
- [Dolt - What is Dolt](https://docs.dolthub.com/introduction/what-is-dolt)
- [Dolt Server Configuration](https://docs.dolthub.com/sql-reference/server/configuration)
- [Dolt Docker Installation](https://docs.dolthub.com/introduction/installation/docker)
- [dolthub/dolt-sql-server Docker Hub](https://hub.docker.com/r/dolthub/dolt-sql-server)
