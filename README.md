# havn

`havn` is a Docker- and Nix-based development environment manager.

## Status

This project is in active development. The specs define the intended CLI
surface, but not every command is fully wired yet.

Project documentation follows an implementation-first policy:

- main setup and usage docs describe behavior that works today
- planned or spec-defined behavior is labeled clearly as planned
- installation guidance prefers the current Go/Make workflow until a separate
  end-user distribution path exists

## Build From Source

Today, the supported path is to build from source:

```bash
make build
make install
```

See `specs/README.md` for the current specification set.

## Troubleshooting

- [Doctor troubleshooting guide](docs/doctor-troubleshooting.md)
