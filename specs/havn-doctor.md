# havn doctor

This is the authoritative doctor contract for `havn`.

Status: Partial

## Ownership

This spec owns:

- what `havn doctor` checks
- how host and container tiers are selected
- prerequisite and skip behavior
- doctor-specific exit semantics and output shapes

Configuration discovery, precedence, and effective-config rules come from
`specs/configuration.md`. General CLI stream separation and error behavior come
from `specs/cli-framework.md`.

## Command Surface

```text
havn doctor [--json] [--verbose] [--all]
```

- `--json` and `--verbose` are persistent CLI flags defined by
  `specs/cli-framework.md`
- `--all` is doctor-specific and extends container selection from the current
  project to all running havn-managed project containers

## Shared Runtime Semantics

Doctor uses the same project context and effective-config semantics as startup
unless this spec explicitly says otherwise.

That means doctor:

- resolves project context from the current working directory when `--all` is
  not set
- interprets global config, project config, discovered `.havn/flake.nix`, and
  environment overrides using `specs/configuration.md`
- uses the same project-identity expectations that startup and shared-Dolt
  wiring use for project-specific checks

Doctor is diagnostic-only. It never creates, modifies, or deletes resources.
Read-only runtime probes such as `SELECT 1` are allowed.

## Check Tiers

Doctor runs two tiers.

### Tier 1: host checks

Host checks always run.

They validate host-side prerequisites and shared infrastructure such as:

- Docker daemon availability
- base image presence
- Docker network presence
- configured named volumes
- global and project config parse/validation
- shared Dolt server health when Dolt is effectively enabled
- project database existence when the current project expects one

### Tier 2: container checks

Container checks run only for relevant running havn-managed project containers.

- default scope: the current project's container, if running
- `--all`: every running havn-managed project container

If no relevant container is running, tier 2 is skipped with an informational
result rather than treated as an error.

Container checks validate runtime wiring such as:

- Nix store accessibility
- dev-shell evaluation
- project mount and config mounts
- SSH agent forwarding
- shared-Dolt connectivity when enabled
- beads health when `.beads/` exists for the project

## Check Catalog

Stable check identifiers:

| Tier | Identifier | Meaning |
|------|------|------|
| host | `docker_daemon` | Docker daemon accessible |
| host | `base_image` | configured base image exists |
| host | `network` | configured Docker network exists |
| host | `volumes` | configured named volumes exist |
| host | `global_config` | global config parses |
| host | `project_config` | project config parses and merged values validate |
| host | `dolt_server` | shared Dolt container is owned by havn, running, and responsive |
| host | `dolt_database` | expected project database exists on shared Dolt |
| container | `nix_store` | `/nix/store` mounted and readable |
| container | `nix_devshell` | configured dev shell evaluates |
| container | `project_mount` | project directory mounted and writable |
| container | `config_mounts` | configured bind mounts present with expected access mode |
| container | `ssh_agent` | SSH agent forwarding is functional |
| container | `dolt_connectivity` | container can reach shared Dolt network path |
| container | `beads_health` | `bd doctor` succeeds or reports its own warnings/errors |

## Prerequisites And Skip Rules

- Checks run in a stable order.
- A check that depends on a failed prerequisite is reported as `skip` with a
  reason.
- If Docker is unavailable, Docker-dependent checks are skipped, but config
  parsing and validation still run.
- `dolt_database` depends on `dolt_server` when a project expects shared Dolt.
- Container-level checks depend on the target container being selected and
  running.

`best-effort` applies across independent checks: one failed check does not stop
other unrelated checks from running.

## What Doctor Verifies Directly

Doctor verifies directly:

- current host and container runtime state
- effective config inputs relevant to the current doctor scope
- presence and responsiveness of shared resources

Doctor may report derived state when that state is part of the published runtime
contract, such as which database name or network doctor expects after effective
config resolution.

Doctor does not redefine separate config semantics of its own.

## Output Contract

### Human output

Default human output groups results by host and container scope and shows the
full outcome, not just failures.

### Verbose output

Verbose output adds details such as versions, resolved paths, timing, and probe
commands while preserving the same pass/warn/error/skip results.

### JSON output

`havn doctor --json` writes a stable JSON object to `stdout`:

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
      "message": "Docker daemon running"
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

Per-check fields:

- `tier`: `host` or `container`
- `container`: present only for container checks
- `name`: stable check identifier
- `status`: `pass`, `warn`, `error`, or `skip`
- `message`: human-readable summary
- `detail`: optional extra context
- `recommendation`: optional remediation guidance

## Exit Codes

- `0`: all checks passed
- `1`: one or more warnings, no errors
- `2`: one or more errors

These doctor-specific exit codes are an exception to the CLI default exit code
rule in `specs/cli-framework.md`.

## First-Run Friendliness

Missing network, volumes, or base image may be expected before first use.
Doctor should report those as non-alarming results where the product contract
defines them as auto-created during normal startup.
