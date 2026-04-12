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

- User-facing docs describe behavior that works today
- Spec-defined behavior that is not fully landed is marked as planned
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
