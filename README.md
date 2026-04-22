# havn

`havn` is a Go CLI for reproducible development environments with Docker and Nix.
It starts (or reuses) one container per project and attaches you directly into a
Nix dev shell.

## What havn does

- Derives a deterministic container name from your project path
- Creates missing shared infrastructure on demand (network, volumes, base image)
- Starts your project container and opens a shell in the configured dev environment
- Supports optional shared Dolt server workflows for projects that use beads

## Status

`havn` is in active development.

- Specs in `specs/` are the normative contract set
- User-facing docs in `README.md` and `docs/` are derivative guides and should be read as current-status guidance
- Support labels use `Implemented`, `Partial`, and `Planned`
- Source builds are the supported installation path right now

For command-level support details, see the support matrix in
[`docs/cli-reference.md`](docs/cli-reference.md).

## Prerequisites

Before using `havn`, install:

- Go (current stable)
- Docker (daemon running)
- Make

## Install (from source)

```bash
make build
make install
```

This builds `bin/havn` and installs the CLI to your Go binary path via
`go install`.

## Quickstart

1. Build and install `havn` from this repository.
2. In any project directory under your home directory, run:

   ```bash
   havn .
   ```

3. On first run, `havn` may create required Docker resources and then attach you
   to the project shell.
4. On later runs, it reuses the existing running container when possible.

### Lifecycle-only startup and debugging flow

- `havn up [path]` is non-interactive lifecycle startup only (create/start/init)
- `havn up --validate [path]` adds required environment validation
- `havn up --prepare [path]` adds validation plus optional environment
  preparation (`havn-session-prepare` when provided)
- debugging-first workflow: run `havn up .` to get the container running, then
  use `havn enter .` for a plain shell

Useful follow-up commands:

```bash
havn list
havn stop --all
havn doctor
```

## Documentation

- [CLI reference](docs/cli-reference.md)
- [Configuration guide](docs/configuration-guide.md)
- [Dolt and beads guide](docs/dolt-beads-guide.md)
- [Doctor troubleshooting guide](docs/doctor-troubleshooting.md)
- [Specs index](specs/README.md)
