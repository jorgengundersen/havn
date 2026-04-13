# AUDIT_REPORT

## Executive summary

This audit synthesis combines nine completed slices (`havn-g56.1` through `havn-g56.9`) and finds that `havn` has a mostly coherent intended architecture and a recognizable implementation, but the published contract is currently more reliable in some subsystems than others.

The strongest areas are the broad CLI shape, the Docker wrapper, and pure contract domains such as config parsing, mount resolution, and naming (`internal/config/*_test.go`, `internal/mount/resolve_test.go:20`, `internal/name/derive_test.go:12`, `internal/docker/integration_test.go:27`). The weakest areas are the config-to-runtime boundary, shared Dolt safety/fidelity, and `havn doctor`, where documented behavior, user-facing docs, and actual runtime behavior diverge in user-visible ways (`internal/cli/config.go:62`, `internal/docker/container.go:53`, `internal/dolt/manager.go:78`, `internal/cli/doctor.go:36`).

The main pattern across the slices is not missing top-level features, but incomplete contract execution: values are resolved but not applied, checks exist but are not run against effective state, and docs/specs duplicate ownership of behavior and have drifted. That leaves maintainers with three immediate needs: tighten the canonical spec surface, align runtime behavior to the published contract, and raise end-to-end confidence around the shipped CLI and shared Dolt paths.

## Audit scope

This synthesis covers:

- Spec and docs coherence, including support-status claims and command/config contracts (`havn-g56.1`)
- CLI and configuration behavior against published contracts (`havn-g56.2`)
- Root command startup, runtime assembly, lifecycle, mounts, ports, and attach behavior (`havn-g56.3`)
- Shared Dolt server lifecycle, ownership, migration, and data-safety behavior (`havn-g56.4`)
- `havn doctor` behavior, output contract, and check fidelity (`havn-g56.5`)
- Architectural layering, dependency boundaries, and code-standards alignment (`havn-g56.6`)
- Test coverage and confidence by subsystem (`havn-g56.7`)
- CI, merge gates, hooks, and portability of quality enforcement (`havn-g56.8`)
- Security- and containment-relevant behavior across mounts, ports, doctor, and Dolt (`havn-g56.9`)

## Method

The underlying slices used spec-to-code and docs-to-code comparison across the published contract set, user-facing docs, CLI wiring, runtime adapters, container lifecycle code, Dolt integration, doctor checks, tests, and CI configuration. Evidence was taken from repository sources and implementation paths, with line-number citations where available, and limited code verification where docs/specs made explicit claims about current behavior.

## Tiered working-version readiness assessment

This section answers the practical question "can I start using `havn` for myself yet?" and then grades readiness upward.

### Tier definitions

- **Tier 0 — Minimum personal-use readiness:** You can reliably use `havn` yourself for a basic dev loop (start/attach, shell entry, stable project container identity) without hitting known contract gaps in common paths.
- **Tier 1 — Comfortable day-to-day personal-use readiness:** Personal use remains reliable across common non-trivial workflows (ports, config overlays, doctor verification, normal stop/list flows) without requiring workarounds.
- **Tier 2 — Collaborator/team-use readiness:** Multiple contributors can share expectations and troubleshoot consistently, with stable CLI contracts, accurate diagnostics, and safe shared-state behavior.
- **Tier 3 — Broader release readiness:** The project can be recommended beyond close collaborators with high confidence in safety, correctness, docs/spec fidelity, and boundary-level test coverage.

### Current tier status

- **Tier 0 (minimum personal-use): PARTIAL / not yet satisfied as a dependable baseline.**
  The core start/attach workflow is mostly present, but key runtime and safety-affecting contract gaps still appear in common paths (for example dropped mount read-only intent, incomplete port-to-runtime propagation, and misleading doctor/effective-config behavior), which makes baseline personal use inconsistent across projects and configurations (`internal/container/start.go:103`, `internal/cli/adapters.go:91`, `internal/docker/container.go:53`, `internal/cli/doctor.go:36`, `internal/doctor/container_checks.go:139`).
- **Tier 1 (comfortable personal-use): NOT satisfied.**
  Non-trivial day-to-day workflows are still unreliable, especially around config fidelity and lifecycle/inspection commands (`internal/cli/config.go:62`, `internal/container/list.go:20`, `internal/cli/stop.go:50`).
- **Tier 2 (collaborator/team-use): NOT satisfied.**
  Divergent docs/contracts, doctor fidelity gaps, and uneven shared-Dolt safety make cross-user trust and troubleshooting inconsistent (`docs/cli-reference.md:82`, `specs/cli-framework.md:178`, `internal/cli/doctor.go:105`, `internal/dolt/migrate.go:53`).
- **Tier 3 (broader release): NOT satisfied.**
  Broader release confidence is blocked by unresolved safety/fidelity issues and insufficient boundary-confidence at the shipped CLI surface (`internal/dolt/manager.go:78`, `internal/dolt/migrate.go:166`, `internal/cli/root_test.go:33`).

### Explicit answer: minimum personal-use readiness

`havn` is **not yet ready for minimum personal use as a dependable default**. It is close in core shape, and some narrow project setups can work today, but the current known runtime-fidelity and safety gaps are still significant enough that "it works for me" depends too much on project specifics and operator workarounds.

### Next tier and concrete blockers

Current practical position is just below **Tier 0**. The next tier to satisfy is **Tier 0 (minimum personal-use readiness)**.

Blockers to move from current state to Tier 0:

1. **Make resolved config authoritative at runtime** (ports, boolean/default merge correctness, and effective-config parity between startup and inspection) (`internal/config/resolve.go:148`, `internal/cli/adapters.go:94`, `internal/cli/config.go:62`).
2. **Preserve mount safety semantics end-to-end** (propagate and enforce read-only mount intent through adapter and Docker layers) (`internal/mount/resolve.go:134`, `internal/cli/adapters.go:91`, `internal/docker/container.go:313`).
3. **Align doctor with effective runtime state** (shared config resolution/name derivation and correct check implementations for mounts/SSH/config validation) (`internal/cli/doctor.go:36`, `internal/cli/doctor.go:89`, `internal/doctor/container_checks.go:179`, `internal/doctor/host_checks.go:258`).
4. **Fix shared-Dolt startup/data-safety correctness on default paths** (startup sequencing, DB defaulting, and safe import/export behavior) (`internal/dolt/manager.go:77`, `internal/dolt/setup.go:31`, `internal/dolt/migrate.go:53`, `internal/dolt/migrate.go:166`).
5. **Tighten user-visible contract consistency** between authoritative specs and derivative docs for currently shipped behavior (`specs/configuration.md`, `specs/cli-framework.md`, `specs/havn-doctor.md`, `docs/cli-reference.md:82`).

Once Tier 0 is satisfied, the next advancement target becomes **Tier 1 (comfortable day-to-day personal-use readiness)**, primarily by hardening stop/list/config/doctor behavior under normal non-trivial workflows and adding stronger end-to-end boundary confidence.

## Findings by domain

### 1. Contract ownership and documentation drift

Contributing slices: `havn-g56.1`, `havn-g56.2`, `havn-g56.8`

- The spec corpus lacks a single canonical config spec even though architecture guidance says config layering and precedence should live there; in practice, `specs/havn-overview.md` is carrying detailed config-contract ownership that `specs/README.md` does not declare, which increases drift risk (`specs/architecture-principles.md` Section 8, `specs/README.md`, `specs/havn-overview.md` Section Configuration).
- Published flag semantics are internally inconsistent: `specs/havn-overview.md` documents runtime flags such as `--shell`, `--env`, `--cpus`, `--memory`, `--port`, `--no-dolt`, and `--image` as global, while `specs/cli-framework.md` says only `--json`, `--verbose`, and `--config` are persistent/global; the code follows the narrower model (`specs/havn-overview.md:83`, `specs/cli-framework.md:178`, `internal/cli/root.go:175`, `internal/cli/root.go:179`).
- User-facing docs overstate support for several commands. `docs/cli-reference.md` marks `havn completion`, `havn config show`, and `havn doctor` as implemented/trustworthy beyond what the code currently does (`docs/cli-reference.md:82`, `docs/cli-reference.md:84`, `docs/cli-reference.md:86`, `internal/cli/root.go:187`, `internal/cli/root.go:198`, `internal/cli/config.go:62`, `internal/cli/doctor.go:36`).
- Shared Dolt behavior is documented in more than one place and has already drifted, especially around `havn dolt status` payload semantics and `.no-sync` behavior visibility (`specs/shared-dolt-server.md` Section Dolt lifecycle, `specs/havn-overview.md` Section JSON output, `docs/dolt-beads-guide.md:68`).

### 2. Configuration resolution and CLI contract fidelity

Contributing slices: `havn-g56.1`, `havn-g56.2`, `havn-g56.3`, `havn-g56.5`

- `havn config show` does not satisfy its published “effective config” contract. It ignores some input sources, does not resolve `.havn/flake.nix`, does not accept root runtime flag overrides, emits a flat `source` map rather than mirrored provenance, and omits `[environment]` from the output (`specs/havn-overview.md:184`, `specs/havn-overview.md:239`, `docs/configuration-guide.md:141`, `internal/cli/config.go:27`, `internal/cli/config.go:62`, `internal/cli/config.go:75`, `internal/config/resolve.go:20`, `internal/config/flake.go:15`, `internal/config/config.go:26`).
- Several resolved config values never reach runtime. Ports are normalized and validated but never passed through container creation, so documented SSH/service publishing does not occur (`specs/havn-overview.md:93`, `specs/havn-overview.md:385`, `internal/cli/root.go:268`, `internal/config/resolve.go:148`, `internal/cli/adapters.go:94`, `internal/docker/container.go:53`).
- Config merge behavior is incomplete for booleans and defaults. `mounts.ssh.*` is not fully overlaid, `dolt.enabled` only overrides when the higher-precedence value is `true`, and the documented defaulting of `dolt.database` to the project directory name is not consistently derived before setup consumes it (`specs/havn-overview.md:98`, `specs/havn-overview.md:324`, `specs/havn-overview.md:355`, `internal/config/resolve.go:54`, `internal/config/resolve.go:95`, `internal/config/resolve.go:107`, `internal/dolt/setup.go:31`).
- The CLI has additional user-visible contract gaps: `havn build` does not accept the documented build-time `--image` override; `havn list` includes stopped containers and the shared `havn-dolt` container rather than only running project containers; `HAVN_SSH_PORT` accepts Docker mapping syntax rather than the documented single port form; and `havn stop --all --json` can return success on partial failure (`docs/cli-reference.md:13`, `docs/cli-reference.md:46`, `specs/havn-overview.md:164`, `specs/havn-overview.md:482`, `internal/cli/build.go:50`, `internal/container/list.go:20`, `internal/docker/container.go:195`, `internal/dolt/manager.go:88`, `internal/config/resolve.go:171`, `internal/config/validate.go:43`, `internal/cli/stop.go:42`, `internal/cli/stop.go:50`).

### 3. Runtime lifecycle, mounts, ports, and containment

Contributing slices: `havn-g56.3`, `havn-g56.9`

- The top-level startup path is largely present: project path resolution under `$HOME`, container naming, attach behavior, shared base-image build contract, and shell exit-code propagation are all recognizable and mostly aligned (`specs/havn-overview.md:50`, `specs/havn-overview.md:56`, `specs/base-image.md:19`, `internal/cli/root.go:152`, `internal/container/start.go:103`, `internal/container/build.go:34`).
- Mount read-only intent is lost before Docker sees it. Mount resolution correctly computes `ReadOnly` for config mounts, SSH agent forwarding, and `authorized_keys`, but the adapter and Docker layer drop that bit, turning intended read-only binds into writable mounts (`internal/mount/types.go:7`, `internal/mount/resolve.go:134`, `internal/mount/resolve.go:146`, `internal/cli/adapters.go:91`, `internal/docker/container.go:313`, `specs/havn-overview.md:310`, `specs/havn-overview.md:556`, `specs/base-image.md:131`).
- Host/container path trust is therefore broader than intended. Absolute host config mounts are allowed, target derivation can place them outside the normal home path inside the container, and the dropped read-only bit turns what should be constrained overlays into writable ones (`internal/cli/root.go:224`, `specs/havn-overview.md:50`, `specs/havn-overview.md:311`, `internal/mount/resolve.go:78`, `internal/mount/resolve.go:108`, `internal/mount/resolve.go:112`).
- Startup self-healing is not classifying failure correctly. Network and volume ensure paths treat any inspect failure as “missing” and try to recreate resources, which can hide daemon or permission failures; `sshd` init errors are ignored and startup can continue into attach despite the documented abort-on-init-failure contract (`specs/havn-overview.md:447`, `specs/havn-overview.md:474`, `internal/container/start.go:120`, `internal/container/start.go:159`, `internal/volume/manager.go:35`).
- Existing stopped-container behavior also diverges: after inspect finds a stopped container, startup can fall through to create rather than start/remove, risking name conflicts instead of performing a documented recovery path (`specs/havn-overview.md:658`, `internal/container/start.go:102`, `internal/container/start.go:112`, `internal/container/start.go:121`).

### 4. Shared Dolt server fidelity and data safety

Contributing slices: `havn-g56.1`, `havn-g56.4`, `havn-g56.9`

- The shared Dolt surface exists, including CLI commands, env injection, and import/export primitives, but fidelity to the documented contract is uneven (`internal/cli/dolt.go:15`, `internal/dolt/setup.go:35`, `internal/dolt/database.go:31`, `internal/dolt/migrate.go:29`).
- Direct `havn dolt start` ignores configured Dolt settings and always starts from defaults, violating documented precedence for image, port, and network (`internal/cli/dolt.go:46`, `specs/havn-overview.md:98`, `specs/shared-dolt-server.md:389`).
- Normal startup still lacks reliable database defaulting. The docs/specs say `dolt.database` defaults to the project directory name, but setup and migration detection use the field verbatim, which can lead to empty database creation attempts (`specs/shared-dolt-server.md:203`, `docs/dolt-beads-guide.md:42`, `internal/dolt/setup.go:31`, `internal/dolt/detect.go:23`).
- New-server provisioning is structurally broken: startup tries to copy `config.yaml` into `havn-dolt` before the container exists, so the documented config-seeding flow cannot work correctly against Docker (`internal/dolt/manager.go:77`, `internal/dolt/manager.go:82`, `internal/docker/copy.go:14`, `specs/shared-dolt-server.md:78`).
- Migration detection is implemented but not wired into normal project startup, so the promised “existing beads database found” notice never appears (`specs/shared-dolt-server.md:559`, `internal/dolt/detect.go:22`, `internal/container/start.go:200`).
- Readiness and ownership checks are incomplete. Readiness polling is skipped when `havn-dolt` is already running, and ownership checks are enforced only for `Start`, not for other commands such as `stop`, `databases`, `drop`, `connect`, `export`, and parts of import/export handling (`specs/shared-dolt-server.md:139`, `specs/shared-dolt-server.md:142`, `internal/dolt/manager.go:67`, `internal/dolt/manager.go:124`, `internal/dolt/database.go:18`, `internal/dolt/migrate.go:75`).
- Import/export paths are not safe enough for persistent data. `--force` import is not a true overwrite flow, import warns about `project_id` mismatch only after writing state, and export untars directly into the destination without atomic staging or overwrite protection, allowing partial state after failure (`specs/shared-dolt-server.md:474`, `specs/shared-dolt-server.md:480`, `internal/dolt/migrate.go:44`, `internal/dolt/migrate.go:53`, `internal/dolt/migrate.go:67`, `internal/dolt/migrate.go:90`, `internal/dolt/migrate.go:123`, `internal/dolt/migrate.go:166`).
- The no-auth, network-isolated shared-server model appears deliberate and the reserved `BEADS_DOLT_*` env story is correctly enforced, but SQL/database-name handling is still weaker than it should be because names are interpolated without corresponding validation guarantees at the Dolt boundary (`specs/shared-dolt-server.md:72`, `specs/shared-dolt-server.md:274`, `internal/dolt/setup.go:46`, `internal/dolt/database.go:32`, `internal/config/validate.go:16`, `internal/container/start.go:245`).

### 5. `havn doctor` correctness and trustworthiness

Contributing slices: `havn-g56.1`, `havn-g56.5`, `havn-g56.7`, `havn-g56.9`

- `havn doctor` has the right overall skeleton: check identifiers, sequential execution, skip behavior, timeouts, and report/JSON structure are mostly aligned with the spec (`specs/havn-doctor.md:308`, `internal/doctor/runner.go:43`, `internal/doctor/format.go:92`).
- The command does not operate on effective runtime config. It hardcodes `config.Default()` and a fixed project-config path instead of using the same merged config flow as startup, so checks for image, network, volumes, project config, and Dolt state can be misleading (`internal/cli/doctor.go:36`, `internal/cli/root.go:231`, `internal/doctor/host_checks_builder.go:11`, `internal/doctor/host_checks.go:258`).
- `--all` is not trustworthy because it enumerates every `managed-by=havn` running container, including `havn-dolt`, and then evaluates project checks using current-directory state rather than per-container effective context (`internal/cli/doctor.go:41`, `internal/cli/adapters.go:292`, `internal/dolt/manager.go:82`).
- Container-name derivation is duplicated and inconsistent with startup, which can make doctor miss the real project container for sanitized names and silently skip container checks (`internal/cli/doctor.go:89`, `internal/container/start.go:269`, `internal/name/path.go:12`, `internal/name/derive.go:17`).
- Several concrete checks are materially wrong: `config_mounts` uses raw config entries rather than resolved in-container mount targets and modes, `ssh_agent` uses literal `$SSH_AUTH_SOCK` in a non-shell exec, and project config is only parsed rather than merged and validated as specified (`specs/havn-doctor.md:88`, `specs/havn-doctor.md:146`, `specs/havn-doctor.md:154`, `internal/doctor/container_checks.go:139`, `internal/doctor/container_checks.go:179`, `internal/doctor/host_checks.go:258`, `internal/docker/exec.go:38`).
- Warning and error outcomes also trigger the CLI’s generic error printer, so doctor emits an extra stderr error payload/message not described by the spec, weakening the reliability of its output contract in both human and JSON mode (`internal/cli/doctor.go:105`, `internal/cli/root.go:42`, `internal/cli/errors.go:116`).

### 6. Architecture and code-structure alignment

Contributing slices: `havn-g56.6`

- The repository broadly follows the intended shape: `cmd/havn` is small, packages are mostly domain-first, and Docker SDK imports are kept inside `internal/docker` rather than spread through domain packages (`specs/architecture-principles.md` Section 4, Section 7; `specs/code-standards.md` Section 1, Section 4).
- The main structural drift is a too-thick CLI boundary. `internal/cli` combines command construction, dependency wiring, logger setup, path policy, config loading/merge, flake resolution, adapter translation, and some domain-adjacent logic, which works against the “thin command / one nameable job” goal (`internal/cli/root.go:95`, `internal/cli/root.go:203`, `internal/cli/root.go:231`, `internal/cli/build.go:89`, `internal/cli/config.go:53`, `internal/cli/dolt.go:313`, `internal/cli/volume.go:34`, `internal/cli/doctor.go:61`).
- User-facing CLI code still knows Docker too directly. Concrete Docker-backed services and Docker-specific error formatting live inside `internal/cli`, which weakens havn-native error and dependency isolation (`internal/cli/root.go:59`, `internal/cli/build.go:22`, `internal/cli/adapters.go:38`, `internal/cli/errors.go:8`, `internal/cli/errors.go:54`).
- Strong resource types and structured logging exist but are not consistently used across cross-package interfaces, leaving many important contracts on raw strings and reducing the intended architectural signal to future contributors (`internal/name/types.go:4`, `internal/container/start.go:18`, `internal/dolt/backend.go:28`, `internal/doctor/backend.go:13`).
- There are also local duplications that already create behavioral divergence, such as doctor’s separate container-name derivation and volume inspection failure collapsing (`internal/cli/doctor.go:91`, `internal/name/path.go:12`, `internal/name/derive.go:17`, `internal/volume/manager.go:25`).

### 7. Test coverage and merge-gate confidence

Contributing slices: `havn-g56.7`, `havn-g56.8`

- Test confidence is high in pure contract domains and medium-high in the Docker wrapper. Config, naming, mount resolution, TOML parsing, validation, flake resolution, and Docker wrapper integration are all meaningfully exercised (`internal/config/config_test.go:15`, `internal/config/resolve_test.go:11`, `internal/config/validate_test.go:12`, `internal/mount/resolve_test.go:20`, `internal/name/path_test.go:12`, `internal/docker/integration_test.go:27`, `internal/docker/integration_test.go:203`).
- Confidence is only medium in CLI/lifecycle behavior because tests are largely in-process with Cobra/fakes rather than true shipped-binary end-to-end checks of stdout/stderr and exit-code contracts (`specs/cli-framework.md` Section 8, `internal/cli/root_test.go:33`, `internal/cli/config_test.go:19`, `internal/cli/stop_test.go:37`, `internal/container/start_test.go:141`).
- Shared Dolt has broad fake-backed coverage but no live integration against real Docker/Dolt behavior, leaving readiness polling, SQL parsing, import/export, and migration-warning paths under-validated (`internal/dolt/manager_test.go:16`, `internal/dolt/setup_test.go:15`, `internal/dolt/migrate_test.go:18`, `internal/dolt/database_test.go:13`).
- `doctor` has good runner/formatter tests but weak behavioral confidence precisely where the implementation diverges from spec, because the tests do not prove effective-config loading, resolved mount verification, or correct container/runtime matching (`internal/doctor/runner_test.go:31`, `internal/doctor/format_test.go:47`, `internal/cli/doctor.go:36`, `internal/doctor/container_checks.go:139`).
- CI and branch-protection intent are real: GitHub Actions runs `make check` plus Docker-backed integration tests, and repo settings declare both required on `main` (`.github/workflows/ci.yml:8`, `.github/settings.yml:2`).
- Merge confidence is still weaker than the docs imply because `make check` rewrites the workspace before validating it, local hooks do not enforce the full formatting/import discipline that CI does, and the hook toolchain is not fully pinned from Go alone (`Makefile:15`, `Makefile:22`, `lefthook.yml:6`, `specs/quality-gates.md:8`, `CONTRIBUTING.md:7`, `go.mod:5`).
- The documented build/test portability story is also not fully aligned with actual enforcement: docs say required merge checks include `go build ./...`, but CI builds only `./cmd/havn`; local integration tests may self-skip, while CI hard-fails on Docker availability before tests can do so (`specs/test-standards.md:376`, `specs/quality-gates.md:21`, `Makefile:3`, `internal/docker/integration_test.go:449`, `.github/workflows/ci.yml:36`).

## Cross-cutting themes

### Canonical contract ownership is unclear

Repeated drift traces back to split ownership between overview specs, dedicated subsystem specs, and user-facing docs. Config behavior, global-flag semantics, and shared Dolt status are each documented in more than one place and no longer fully agree (`specs/havn-overview.md:83`, `specs/cli-framework.md:178`, `specs/shared-dolt-server.md` Section Dolt lifecycle, `docs/cli-reference.md:86`).

### Values are often resolved but not enforced at runtime

Several bugs share the same pattern: configuration or intent is computed correctly and then dropped before execution. That appears in port publishing, mount read-only mode, doctor’s use of effective config, and shared Dolt startup/config-seeding (`internal/config/resolve.go:148`, `internal/cli/adapters.go:91`, `internal/docker/container.go:53`, `internal/cli/doctor.go:36`, `internal/dolt/manager.go:78`).

### Safety checks exist, but classification and sequencing are weak

The code often has guardrails, but important paths misclassify errors or perform checks too late. Examples include network/volume self-heal treating all inspect failures as “missing,” import warning after persistent Dolt writes, and doctor reporting plus extra generic error output (`internal/container/start.go:159`, `internal/volume/manager.go:35`, `internal/dolt/migrate.go:67`, `internal/cli/doctor.go:105`).

### Thin-boundary architectural intent is only partially realized

The most persistent implementation debt sits at the CLI boundary, where config resolution, runtime translation, path policy, and Docker-coupled behavior accumulate. That same concentration shows up in contract drift, duplicated derivation logic, and difficulty reusing the same effective-state primitives between startup, doctor, and direct subsystem commands (`internal/cli/root.go:231`, `internal/cli/doctor.go:61`, `internal/cli/adapters.go:38`).

### Confidence is highest below the published user contract

Tests are strongest in pure libraries and the Docker wrapper, but weakest where end users feel the system: the shipped CLI surface, `doctor`, and shared Dolt lifecycle. The result is a gap between internal confidence and external contract confidence (`internal/config/resolve_test.go:11`, `internal/docker/integration_test.go:27`, `internal/cli/root_test.go:33`, `internal/dolt/manager_test.go:16`, `internal/doctor/runner_test.go:31`).

## Follow-up recommendations

1. **Create a contract-alignment epic around canonical spec ownership.** Add a dedicated config spec, reduce duplicated command contracts in `specs/havn-overview.md`, and make one source authoritative for shared Dolt and support-status claims. This should explicitly reconcile flag scope, `config show`, `doctor`, completion support, and `dolt status` semantics. Evidence: `specs/architecture-principles.md` Section 8, `specs/havn-overview.md:83`, `specs/cli-framework.md:178`, `docs/cli-reference.md:82`, `docs/cli-reference.md:86`.

2. **Fix the config-to-runtime pipeline before broadening features.** Priority items are: port publishing, mount `ReadOnly` propagation, boolean/default merge correctness, `HAVN_SSH_PORT` normalization, and making `havn config show` reflect the same effective config startup uses. Evidence: `internal/config/resolve.go:148`, `internal/cli/adapters.go:91`, `internal/docker/container.go:53`, `internal/config/resolve.go:95`, `internal/cli/config.go:62`.

3. **Treat shared Dolt as a data-safety workstream, not just a command-surface workstream.** Fix startup sequencing, database defaulting, ownership/readiness enforcement, migration detection wiring, and import/export overwrite/rollback semantics before increasing reliance on the feature. Evidence: `internal/cli/dolt.go:46`, `internal/dolt/manager.go:78`, `internal/dolt/setup.go:31`, `internal/dolt/manager.go:124`, `internal/dolt/migrate.go:53`.

4. **Refit `havn doctor` to consume the same effective-state primitives as startup.** This should include shared config loading, shared name derivation, resolved mount checking, proper SSH-agent verification, and output behavior that does not double-report via generic CLI errors. Evidence: `internal/cli/doctor.go:36`, `internal/cli/doctor.go:89`, `internal/doctor/container_checks.go:139`, `internal/doctor/container_checks.go:179`, `internal/cli/errors.go:116`.

5. **Reduce CLI thickness by extracting reusable orchestration services.** Move config resolution, runtime translation, and project-context derivation behind shared services used by root startup, doctor, Dolt commands, and config inspection. This should lower drift and make end-to-end behavior easier to test. Evidence: `internal/cli/root.go:231`, `internal/cli/build.go:89`, `internal/cli/config.go:53`, `internal/cli/doctor.go:61`.

6. **Raise confidence at the published CLI boundary.** Add true end-to-end tests for the built binary’s stdout/stderr/exit-code behavior, live shared Dolt integration coverage, and doctor behavioral tests against effective config and resolved mounts. Evidence: `specs/cli-framework.md` Section 8, `internal/dolt/manager_test.go:16`, `internal/doctor/runner_test.go:31`, `internal/cli/root_test.go:33`.

7. **Tighten merge-gate fidelity and local/CI parity.** Stop validating only a rewritten tree, align local hook enforcement with `make fmt`, and reconcile documented build/test requirements with actual CI behavior. Evidence: `Makefile:15`, `Makefile:22`, `lefthook.yml:6`, `specs/test-standards.md:376`, `.github/workflows/ci.yml:20`.
