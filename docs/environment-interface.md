# havn environment interface

This is a derivative guide for environment authors.

For normative behavior, use `specs/environment-interface.md`.

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

## Command behavior contract

- `havn [path]`
  - Starts or reuses container and attaches interactively.
  - Runs `havn-session-prepare` if available.
  - If prepare runs and exits non-zero, command fails with actionable error.

- `havn up [path]`
  - Starts or reuses container in non-interactive mode.
  - Runs `havn-session-prepare` if available.
  - If prepare runs and exits non-zero, command fails non-zero without attach.

- `havn enter [path]`
  - Enters running container with plain shell semantics.
  - Does not run startup lifecycle preparation.

## Minimal compatible environment example

```nix
{
  outputs = { self, nixpkgs, ... }: {
    devShells.x86_64-linux.default =
      let pkgs = import nixpkgs { system = "x86_64-linux"; };
      in pkgs.mkShell {
        packages = with pkgs; [ git bash ];
      };
  };
}
```

## Example with optional prepare hook

```nix
{
  outputs = { self, nixpkgs, ... }: {
    devShells.x86_64-linux.default =
      let pkgs = import nixpkgs { system = "x86_64-linux"; };
      in pkgs.mkShell {
        packages = with pkgs; [ git bash ];
      };

    apps.x86_64-linux.havn-session-prepare = {
      type = "app";
      program = "${
        (import nixpkgs { system = "x86_64-linux"; }).writeShellScript "havn-session-prepare" ''
          set -eu
          # Optional environment-owned preparation logic goes here.
          exit 0
        ''
      }";
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

## Test and validation guidance

- Use local fixture flakes for contract tests.
- Do not hardcode personal repo refs in authoritative tests.
- Validate at least:
  - required shell exists
  - optional prepare hook missing
  - optional prepare hook success
  - optional prepare hook failure

If this guide and specs differ, specs win.
