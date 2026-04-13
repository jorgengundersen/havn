# havn doctor troubleshooting guide

`havn doctor` is the first command to run when a havn environment is not behaving as expected. This guide is derivative; the normative doctor contract lives in `specs/havn-doctor.md`. Doctor is diagnostic-only: it reports what is wrong and what to do next, but it does not change your system.

## Purpose

- Use `havn doctor` to check host prerequisites and running container health.
- Start with doctor before running ad-hoc `docker` commands so you get havn-specific guidance.
- If a check fails, use the recommendation from doctor first, then rerun doctor to confirm recovery.

## Flags and output modes

```bash
havn doctor [--all] [--verbose] [--json]
```

- Default output is human-readable and grouped by host checks and per-container checks.
- `--all` checks all running havn containers. Without it, doctor checks only the current project container.
- `--verbose` includes details such as versions, paths, timings, and underlying checks.
- `--json` emits machine-readable output for automation and scripts.

## Exit codes

- `0`: all checks passed.
- `1`: one or more warnings (non-blocking issues).
- `2`: one or more errors (broken state requiring action).

## How to read check results

- `pass`: check succeeded.
- `warn`: degraded state, often recoverable without blocking work.
- `error`: broken state that should be fixed before continuing.
- `skip`: check was not run because a prerequisite failed.

## Check tiers

- Tier 1 (host): always runs and validates Docker, image, network, volumes, config, and Dolt host-side checks.
- Tier 2 (container): runs for relevant running containers and validates in-container wiring such as Nix, mounts, SSH agent, and beads health.

## Troubleshooting flows

### Docker daemon check failed

1. Start Docker Desktop or Docker Engine.
2. Confirm daemon access with `docker info`.
3. If access is denied, ensure your user can access Docker (for example membership in the `docker` group).
4. Rerun `havn doctor`.

### Base image check warned

1. Build or rebuild the base image: `havn build`.
2. Rerun `havn doctor`.
3. If it still warns, run `havn build --verbose` and inspect build errors.

### Network or volume checks warned

1. If this is your first run, warnings can be expected.
2. Start a project once with `havn .` so havn can auto-create missing resources.
3. Rerun `havn doctor`.

### Global or project config check failed

1. Open the config file reported by doctor.
2. Fix syntax or invalid values at the indicated field or line.
3. Validate by rerunning `havn doctor`.

### Dolt server check failed

1. Start the shared server: `havn dolt start`.
2. If the container is running but unhealthy, inspect logs: `docker logs havn-dolt`.
3. Retry the server lifecycle: `havn dolt stop && havn dolt start`.
4. Rerun `havn doctor`.

### Container-level Nix or mount checks failed

1. Stop and restart the project container with `havn .`.
2. If mounts still fail, verify the host path exists and permissions are correct.
3. Rerun `havn doctor --verbose` to get the failing path/check details.

### SSH agent check warned

1. Confirm `ssh-agent` is running on the host.
2. Confirm `SSH_AUTH_SOCK` is set on the host.
3. Restart the project container with `havn .` and rerun doctor.

### Beads health warned

1. Run `bd doctor --json` inside the project container for beads-specific diagnostics.
2. Follow the remediation from beads output.
3. Rerun `havn doctor` to confirm container plumbing and beads health are both restored.

## Recommended command sequence

```bash
havn doctor
havn doctor --verbose
havn doctor --json
```

Use this sequence when triaging hard-to-reproduce issues: quick status, deep detail, then structured output for sharing or automation.

## Current partial-support gaps

- `havn doctor` currently reports an extra container-tier skip check (`container_tier`) when no relevant container is running; this check identifier is not part of the published doctor check catalog yet
