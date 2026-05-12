# Project-specific dependencies

Use this guide when a project needs tools, language runtimes, or setup steps
that are specific to that project.

`havn` does not install project dependencies into the base Docker image. Project
specific tools come from the selected Nix dev shell at startup time.

## The model

A havn project declares dependencies with a project-local Nix flake. The flake
must expose the required environment entrypoint:

```text
devShells.<system>.<shell>
```

Common defaults are:

- `<system>`: `x86_64-linux`
- `<shell>`: `default`

`havn` auto-discovers these project-local flakes, in order:

1. `<project>/.havn/flake.nix` as `path:./.havn`
2. `<project>/.havn/environments/default/flake.nix` as
   `path:./.havn/environments/default`

If the flake is at one of those paths and the shell is named `default`, no
project config is required.

## Minimal project flake

Create `<project>/.havn/flake.nix`:

```nix
{
  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs = { nixpkgs, ... }:
    let
      system = "x86_64-linux";
      pkgs = import nixpkgs { inherit system; };
    in
    {
      devShells.${system}.default = pkgs.mkShell {
        packages = with pkgs; [
          git
          bash
          go
          nodejs_22
          python3
        ];
      };
    };
}
```

Then verify:

```bash
havn config show --json
havn up --validate .
havn .
```

Commit both files:

```text
.havn/flake.nix
.havn/flake.lock
```

## Extending a default environment

If you already have a default environment flake and want
`<project>/.havn/flake.nix` to add project-specific tools on top, use
`pkgs.mkShell` with `inputsFrom`.

Example:

```nix
{
  inputs = {
    # Replace this with your shared/default environment flake.
    baseEnv.url = "github:OWNER/DEFAULT-ENV-REPO";

    # Prefer following the base environment's nixpkgs when it exposes one.
    # If the base flake does not expose a nixpkgs input, replace this with an
    # explicit nixpkgs URL instead.
    nixpkgs.follows = "baseEnv/nixpkgs";
  };

  outputs = { nixpkgs, baseEnv, ... }:
    let
      system = "x86_64-linux";
      pkgs = import nixpkgs { inherit system; };
    in
    {
      devShells.${system}.default = pkgs.mkShell {
        # Inherit tools/build inputs from the shared default shell.
        inputsFrom = [
          baseEnv.devShells.${system}.default
        ];

        # Add project-specific tools here.
        packages = with pkgs; [
          postgresql_16
          redis
          nodejs_22
        ];

        shellHook = ''
          echo "project environment ready"
        '';
      };
    };
}
```

Notes:

- `inputsFrom` is the usual way to compose another `mkShell`-style dev shell
  with additional project packages.
- `inputsFrom` inherits shell inputs; do not rely on it to run arbitrary setup
  side effects from the base environment.
- If the base environment has required startup setup, expose or wrap
  `apps.<system>.havn-session-prepare` as shown below.

## Extending a non-default shell

If the base environment shell is named `codex` and you want havn to enter that
shell name, expose and configure the same shell name:

```nix
{
  inputs = {
    baseEnv.url = "github:OWNER/DEFAULT-ENV-REPO";
    nixpkgs.follows = "baseEnv/nixpkgs";
  };

  outputs = { nixpkgs, baseEnv, ... }:
    let
      system = "x86_64-linux";
      pkgs = import nixpkgs { inherit system; };
    in
    {
      devShells.${system}.codex = pkgs.mkShell {
        inputsFrom = [
          baseEnv.devShells.${system}.codex
        ];

        packages = with pkgs; [
          just
          ripgrep
        ];
      };
    };
}
```

Add `<project>/.havn/config.toml`:

```toml
shell = "codex"
```

Because `<project>/.havn/flake.nix` is auto-discovered, you do not need to set
`env` unless the flake lives somewhere else.

## Optional startup preparation

A project flake may expose this optional app:

```text
apps.<system>.havn-session-prepare
```

Use it only for non-interactive, idempotent setup that should run before the
interactive shell starts, such as Home Manager activation or project bootstrap.

If your base environment already exposes `havn-session-prepare`, you can
re-export it:

```nix
apps.${system}.havn-session-prepare =
  baseEnv.apps.${system}.havn-session-prepare;
```

Or wrap it and add project-specific setup:

```nix
apps.${system}.havn-session-prepare =
  let
    basePrepare = baseEnv.apps.${system}.havn-session-prepare.program;
  in
  {
    type = "app";
    program = "${pkgs.writeShellScript "havn-session-prepare" ''
      set -eu

      "${basePrepare}"

      # Project-specific non-interactive setup goes here.
      # Keep this idempotent and prompt-free.
    ''}";
  };
```

Verify the prepare hook with:

```bash
havn up --prepare .
havn .
```

## Agent checklist

When instructing an AI agent to add project-specific dependencies:

1. Do not modify the havn base image for project tools.
2. Create or update `<project>/.havn/flake.nix`.
3. Ensure the flake exposes `devShells.<system>.<shell>`.
4. Add project tools to the dev shell `packages` list.
5. If extending a shared/default shell, use `inputsFrom`.
6. Add `<project>/.havn/config.toml` only when selecting a non-default shell or
   a non-discovered flake path.
7. Add `apps.<system>.havn-session-prepare` only when startup preparation is
   required.
8. Keep preparation non-interactive and idempotent.
9. Run `havn config show --json` and `havn up --validate .`.
10. Commit `.havn/flake.nix`, `.havn/flake.lock`, and `.havn/config.toml` when
    present.

## Related docs

- `environment-interface.md` defines the required and optional flake outputs.
- `configuration-guide.md` explains project config, flake discovery, and
  precedence.
- `cli-reference.md` explains startup flags and validation commands.
