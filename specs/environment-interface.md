# Environment Interface

This is the authoritative contract for how a Nix flake environment integrates
with `havn`.

Status: Partial

Status progression for this contract is `Planned` -> `Partial` ->
`Implemented`.

`Partial` means this contract is authoritative and ratified, but runtime
alignment is still in progress and tracked separately from ratification.

## Ownership

This spec owns:

- required environment entrypoints that `havn` depends on
- optional environment capability entrypoints for startup preparation
- command-surface behavior when optional capabilities are missing or failing
- portability and boundary rules between orchestration and environment content

`havn` orchestration behavior at the CLI boundary remains owned by
`specs/cli-framework.md`. Configuration precedence remains owned by
`specs/configuration.md`.

## Goals

- keep `havn` generic and reusable by anyone
- keep environment definitions independent from `havn` internals
- avoid hardcoded personal repository paths and user-specific assumptions
- expose capabilities via explicit flake outputs

## Required entrypoint

An environment must expose:

- `devShells.<system>.<shell>`

`<system>` is the target platform key used by Nix flake outputs (for example
`x86_64-linux`). `<shell>` is the effective shell name resolved by `havn`
configuration.

If the required shell output is missing for the resolved pair,
commands that execute environment validation or shell handoff fail with a
command error.

## Optional capability entrypoint

An environment may expose:

- `apps.<system>.havn-session-prepare`

This optional app is the environment-owned startup preparation hook. Typical
uses include Home Manager activation or environment-local session bootstrap.

For v1, the capability name `apps.<system>.havn-session-prepare` is reserved
and stable.

When missing, the capability is treated as not provided and startup continues.

When present, `havn` executes it as a non-interactive command and treats
non-zero exit as command failure.

## Command-surface behavior

Capability behavior is command-scoped:

- `havn [path]`
  - runs `havn-session-prepare` when available
  - if prepare fails, exits non-zero and does not hand off to interactive shell
- `havn up [path]`
  - default run remains non-interactive and never attaches
  - default run does not execute `havn-session-prepare`
  - `havn up --validate [path]` validates that
    `devShells.<system>.<shell>` is realizable in non-interactive mode
  - `havn up --prepare [path]` runs `havn-session-prepare` when available
  - `havn up --prepare [path]` performs the same validation before running
    `havn-session-prepare`
  - when `--prepare` is used and prepare fails, exits non-zero with actionable
    command-scoped error
- `havn enter [path]`
  - plain-shell entry only
  - does not run startup preparation capability

## Execution constraints for `havn-session-prepare`

The prepare hook must be safe for startup automation:

- non-interactive (must not require prompts)
- idempotent (safe to run repeatedly)
- deterministic exit semantics (`0` success, non-zero failure)
- when the hook evaluates or builds nested flake targets (for example Home
  Manager activation packages), refresh behavior must be explicit in those
  nested commands rather than assumed from outer invocation context
- when environments expose a user override for nested refresh behavior, refresh
  should remain enabled by default and opt-out via explicit override

`havn` does not define environment-specific side effects beyond invoking this
entrypoint when present.

## Portability boundaries

- `havn` must not hardcode personal repository URLs or local filesystem paths
  for environment discovery or capability invocation.
- Authoritative tests for this contract must use local fixture flakes and must
  not depend on one personal environment repository.
- Environment flakes should prefer flake-relative inputs and portable path
  handling.

## Versioning and compatibility

- This contract is versioned by spec revision.
- Additive entrypoints and additive metadata are non-breaking.
- Renaming or removing required entrypoints is breaking.
- Changing behavior for missing optional capabilities from "skip" to "fail"
  is breaking.
- Ratification closes at `Partial`; full runtime parity is tracked as
  implementation follow-up work.

## Relationship to other specs

- `specs/cli-framework.md` owns command tree, flags, stream separation, CLI
  output, and command error framing.
- `specs/configuration.md` owns `env` and `shell` discovery and precedence that
  determine which entrypoints are resolved.
- `specs/havn-overview.md` summarizes this contract at product level and links
  here for normative detail.
