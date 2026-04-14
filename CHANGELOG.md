# Changelog

All notable changes to this project are documented in this file.

The format is based on Keep a Changelog, and this project follows Semantic Versioning.

## [0.1.0] - 2026-04-14

Initial public release of `havn`, a Go CLI for reproducible development environments with Docker and Nix.

### Added

- Project-oriented start/attach workflow via `havn [path]`, including deterministic container naming.
- Automatic provisioning of required runtime infrastructure (base image, network, and shared volumes).
- Core CLI commands: `havn list`, `havn stop`, `havn build`, `havn volume list`, and `havn config show`.
- Health diagnostics with `havn doctor` for host and container checks.
- Shared Dolt workflow commands under `havn dolt` (start, stop, status, databases, drop, connect, import, export).
- JSON output mode (`--json`) and verbose diagnostics (`--verbose`) across command surfaces.

### Changed

- Consolidated CLI/runtime orchestration and Docker adapter boundaries to reduce command-surface drift.
- Tightened repository quality gates and boundary-confidence coverage for shipped CLI behavior.
- Clarified documentation authority and support-status labeling across docs and specs.

### Fixed

- Base image build robustness when host UID/GID collides with existing accounts/groups in Ubuntu base images.
- Nix command invocation now enables required experimental features for `nix develop` flows.
- Docker image build path compatibility improvements for Desktop/containerd environments.
