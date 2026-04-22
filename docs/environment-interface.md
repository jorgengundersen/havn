# havn environment interface

This is a derivative guide for environment authors.

For normative behavior, use `specs/environment-interface.md`.

Status alignment: the authoritative interface contract is ratified at
`Status: Partial`; remaining gaps are runtime-alignment work.

Goal: keep orchestration (`havn`) and environment definitions independent, with
clear and portable integration points.

## Design principles

- `havn` stays generic and does not assume personal repository layout.
- Environment flakes declare capabilities through explicit outputs.
- Required interface stays minimal.
- Optional lifecycle behavior is opt-in.
- No hardcoded user-specific paths in authoritative examples or tests.

## Required interface

An environment must expose a dev shell that `havn` can enter.

- Required output: `devShells.<system>.<shell>`
- Typical values:
  - `<system>`: `x86_64-linux`, `aarch64-linux`, ...
  - `<shell>`: usually `default` unless configured otherwise

If this output is missing for the resolved `<system>/<shell>`, `havn` startup
is a contract error.

## Optional interface

An environment may expose an optional lifecycle preparation entrypoint:

- Optional output: `apps.<system>.havn-session-prepare`

When present, this entrypoint is where environment-owned session preparation
happens (for example Home Manager activation or shell-local bootstrap).

When absent, `havn` must treat this as capability not provided, not as a hard
failure.

## For coding agents

Use this checklist when generating or modifying havn-compatible environments.

- Do define `devShells.<system>.<shell>` first. This is the required entrypoint.
- Do treat `apps.<system>.havn-session-prepare` as optional capability only.
- Do make `havn-session-prepare` non-interactive and idempotent.
- Do make `havn-session-prepare` resilient to minimal env contexts (do not assume
  `USER` is set; derive identity safely when needed).
- Do use flake-relative inputs and portable paths.
- Do not hardcode one user's home directory, username, or repo location.
- Do not require prompts, manual confirmations, or one-time local state.

Authoring order for agents:

1. Create a valid `devShells.<system>.<shell>` output.
2. Add optional `apps.<system>.havn-session-prepare` only when startup
   preparation is needed.
3. Ensure prepare exits `0` on success and non-zero on failure.
4. Verify behavior through local fixture-based checks, not personal repo refs.

## Command behavior contract

- `havn [path]`
  - Starts or reuses container and attaches interactively.
  - Validates required `devShells.<system>.<shell>` resolution before shell
    handoff.
  - Runs `havn-session-prepare` if available.
  - If validation or prepare fails, command fails non-zero with actionable
    error.

- `havn up [path]`
  - Default run starts or reuses container in non-interactive mode.
  - Default run does not execute `havn-session-prepare`.
  - `havn up --validate [path]` validates required
    `devShells.<system>.<shell>` realizability in non-interactive mode.
  - `havn up --prepare [path]` implies validation, then runs
    `havn-session-prepare` when available.
  - For `--validate`/`--prepare`, failures are non-zero and non-interactive.

- `havn enter [path]`
  - Enters running container with plain shell semantics.
  - Does not run startup lifecycle preparation.

## Copy/paste starter template

Use this as a starter and replace `x86_64-linux` and `default` as needed.

### Minimal compatible variant (required only)

```nix
{
  outputs = { nixpkgs, ... }: {
    # Required: devShells.<system>.<shell>
    devShells.x86_64-linux.default =
      let
        pkgs = import nixpkgs { system = "x86_64-linux"; };
      in
      pkgs.mkShell {
        packages = with pkgs; [ git bash ];
      };
  };
}
```

### Optional prepare capability variant

```nix
{
  outputs = { nixpkgs, ... }: {
    # Required: devShells.<system>.<shell>
    devShells.x86_64-linux.default =
      let
        pkgs = import nixpkgs { system = "x86_64-linux"; };
      in
      pkgs.mkShell {
        packages = with pkgs; [ git bash ];
      };

    # Optional: apps.<system>.havn-session-prepare
    apps.x86_64-linux.havn-session-prepare =
      let
        pkgs = import nixpkgs { system = "x86_64-linux"; };
      in
      {
        type = "app";
        program = "${pkgs.writeShellScript "havn-session-prepare" ''
          set -eu
          # Optional environment-owned preparation logic goes here.
          exit 0
        ''}";
      };
  };
}
```

## Portability rules for environment authors

- Do not hardcode paths to one user's home directory.
- Do not require one specific repository location on disk.
- Prefer flake-relative paths and inputs.
- If external repos are referenced, pin revisions when reproducibility matters.
- Keep `havn-session-prepare` idempotent and non-interactive.

## Validation guidance for contributors and agents

Authoritative validation for this interface is fixture-backed and local-first:

- Authoritative contract matrix tests live in `internal/container` and
  `internal/cli`.
- Authoritative tests use local fixture flakes only.
- Authoritative tests do not hardcode personal paths or personal repository
  references.

Contract scenario coverage must include:

- missing required `devShells.<system>.<shell>`
- missing optional `apps.<system>.havn-session-prepare`
- successful optional `apps.<system>.havn-session-prepare`
- failing optional `apps.<system>.havn-session-prepare`

Command-surface semantics for these scenarios should be validated across
`havn`, `havn up`, and `havn enter` where applicable.

For local validation, use:

```bash
make check
make test-boundary-confidence
make test-integration
```

`make check` is the baseline gate. Use boundary-confidence and integration
targets when changing behavior around the environment contract.

### Optional cross-repo smoke checks

The repository also provides a manual smoke workflow at
`.github/workflows/smoke-cross-repo.yml` for checking external environment
repositories against this contract.

- It is intentionally non-authoritative and does not replace fixture-backed
  contract tests.
- It is manually triggered (`workflow_dispatch`) and therefore does not run on
  every push or pull request.
- Required merge checks remain `quality-gates`, `integration-tests`, and
  `boundary-confidence` from `.github/workflows/ci.yml`.
- Use smoke results as compatibility signals only; when smoke and fixture tests
  disagree, follow `specs/environment-interface.md` and fix authoritative tests
  first.

If this guide and specs differ, specs win.
