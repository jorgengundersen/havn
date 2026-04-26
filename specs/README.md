# Specs

This directory is the normative contract set for `havn`.

## Spec Governance

### Authority levels

- **Authoritative spec**: normative contract for behavior, ownership, and invariants. When code or user docs disagree with an authoritative spec, the spec wins and the drift should be fixed or called out.
- **Derivative doc**: explanatory or task-oriented guidance in `docs/` or `README.md`. Derivative docs summarize current behavior and point back to the authoritative spec for exact contracts.
- **Overview spec**: product framing and reading order only. It links to subsystem authorities instead of restating their detailed contracts.

### Support-status labels

Use these labels on behavior-bearing specs and derivative docs when support status matters:

- **Implemented**: intended contract is shipped and is the current supported behavior.
- **Partial**: contract is authoritative, but current implementation does not yet satisfy all of it. Derivative docs must call out the gap instead of presenting the whole contract as landed.
- **Planned**: intended surface or behavior is accepted into the spec corpus but is not yet shipped.
- **Derivative**: explanatory guidance only; not a support claim on its own.

Meta specs such as coding or testing standards do not need support-status labels unless they describe user-visible behavior.

### Adding or changing a major spec

When a new major contract is added:

1. Choose one authoritative owner for the behavior.
2. Add the spec to the index below with its role.
3. Update `specs/havn-overview.md` so the overview points to the new owner.
4. Update derivative docs in `README.md` or `docs/` so they link back to the authoritative spec instead of re-defining the contract.

## Shared Vocabulary

- **persistent flag**: a CLI flag defined on the root command and inherited by subcommands.
- **root-only flag**: a flag accepted only by `havn [path]`; it is not inherited by subcommands.
- **effective config**: the fully resolved configuration after applying defaults, discovered config files, environment overrides, and any command-specific runtime overrides that the current command accepts.
- **machine-readable JSON contract**: the stable JSON written to `stdout` for command data, with errors written to `stderr`.
- **best-effort**: continue independent work after one unit fails, then report the full mixed outcome.
- **derivative doc**: user guidance that explains behavior but defers normative detail to an authoritative spec.

## Cross-Spec Invariants

- Configuration discovery, precedence, and provenance are owned by `specs/configuration.md`.
- Startup project-path boundary for `havn [path]` and `havn up [path]` (resolved path must be under the user's home directory) is owned by `specs/configuration.md`.
- `havn doctor` uses the same project context and effective-config rules as startup, and reuses shared-Dolt naming/config expectations for diagnostics only; it never performs provisioning or other mutation.
- CLI stream separation is a cross-command invariant for non-interactive command output: status, logs, and errors go to `stderr`; command data and stable JSON go to `stdout`.
- Interactive attach commands are the explicit exception while the attached subprocess owns the TTY stream (`havn [path]`, `havn enter [path]`, and `havn dolt connect`): separate stderr capture is not guaranteed during that interactive session.
- `specs/havn-overview.md` is never the hidden authority for config, doctor, CLI, or shared-Dolt detail; it points to the owning spec.
- Shared Dolt lifecycle, readiness, ownership, startup provisioning, and shared-server status/databases semantics are owned by `specs/shared-dolt-server.md`.
- Beads data migration and project-identity migration policy are owned by beads tooling/contracts.
- `havn dolt import/export` migration correctness semantics are explicit non-goals for havn-owned shared-Dolt behavior.
- Environment flake integration entrypoints and optional startup capabilities are owned by `specs/environment-interface.md`.

## Spec Index

| Spec | Role |
|------|------|
| [architecture-principles.md](architecture-principles.md) | Foundational engineering principles and architectural constraints |
| [code-standards.md](code-standards.md) | Go-specific implementation conventions |
| [test-standards.md](test-standards.md) | Testing conventions and boundaries |
| [quality-gates.md](quality-gates.md) | Tooling, build, lint, and CI gates |
| [configuration.md](configuration.md) | Authoritative configuration contract: discovery, precedence, merge rules, and config inspection |
| [cli-framework.md](cli-framework.md) | Authoritative CLI contract: command tree, flag scope, output handling, and CLI error behavior |
| [environment-interface.md](environment-interface.md) | Authoritative environment integration contract: required entrypoints, optional startup capability, and portability boundaries |
| [havn-overview.md](havn-overview.md) | Product overview, core workflows, and pointers to authoritative subsystem specs |
| [base-image.md](base-image.md) | Base image and runtime-init contract |
| [havn-doctor.md](havn-doctor.md) | Authoritative doctor contract: checks, tiers, selection rules, and output |
| [shared-dolt-server.md](shared-dolt-server.md) | Authoritative shared-Dolt contract: lifecycle, readiness, ownership, startup provisioning, and shared-server status/databases |
