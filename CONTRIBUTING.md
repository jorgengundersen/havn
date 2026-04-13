# Contributing to havn

Thanks for contributing to `havn`. This guide is the developer onboarding path for local setup, day-to-day workflow, and issue tracking.

## Prerequisites

- Go (latest stable)
- Git
- Docker (required for integration tests and local runtime behavior)
- `bd` (beads CLI for issue tracking)
- `lefthook` (for git hook quality gates)

## Local setup

1. Clone the repository.
2. Install hooks:

   ```bash
   lefthook install --force
   bd hooks install --beads --chain
   ```

3. Build and install locally:

   ```bash
   make build
   make install
   ```

4. Verify the basic quality gate:

   ```bash
   make check
   ```

## Development workflow

Use red/green TDD and keep changes small:

1. Write one failing test for one behavior.
2. Implement the minimum change to make it pass.
3. Refactor while keeping tests green.
4. Repeat.

Follow these standards while implementing:

- `specs/code-standards.md`
- `specs/test-standards.md`
- `specs/quality-gates.md`

## Quality gates

Run these targets before opening a PR:

- `make fmt` - format code and imports
- `make lint` - run static analysis (`golangci-lint`)
- `make test` - run unit tests
- `make test-integration` - run Docker-backed integration tests
- `make test-boundary-confidence` - run boundary-confidence suites for shipped CLI contracts
- `make build` - compile `bin/havn`
- `make check` - run fmt + lint + test + build

Commits trigger hooks. A failing hook blocks the commit.

`integration-tests` and `boundary-confidence` are required merge checks on `main`.

## Repository structure

- `cmd/havn/` - binary entrypoint and top-level wiring
- `internal/cli/` - Cobra command definitions and CLI boundary behavior
- `internal/` domain packages - config, docker wrappers, dolt, mounts, names, volumes
- `internal/ci/` - repository contract tests for docs/spec/workflow guarantees
- `specs/` - authoritative implementation specs
- `docs/` - user and operational guides

## Working with bd issues

This repository uses `bd` for all work tracking.

Typical flow:

```bash
bd ready --json
bd update <id> --claim --json
# implement + test
bd close <id> --reason "Completed" --json
```

Rules:

- Always track work in `bd`; do not maintain markdown TODO lists.
- Use `--json` in scripted/agent workflows.
- Link discovered follow-up work with `discovered-from:<parent-id>` dependencies.

## Pull request workflow

1. Keep your branch up to date (`git pull --rebase`).
2. Run `make check`, `make test-integration`, and `make test-boundary-confidence`.
3. Commit using a conventional commit style (`feat:`, `fix:`, `docs:`, `test:`, `refactor:`, `chore:`).
4. Push your branch and open a PR with a clear summary of intent and behavior changes.
5. Reference and close related `bd` issues in the PR and issue notes.

Prefer implementation-first documentation updates: document what works today, and clearly label planned behavior when needed.
