# Base Image Implementation Spec

Concrete contract for the havn base image, Docker build context, and runtime
init assumptions referenced by `havn-overview.md`.

## 1. Scope

This spec defines:

- the repository build context used for the havn base image
- the image contract consumed by `havn build` and startup auto-build
- UID/GID handling for the `devuser` account
- the runtime assumptions behind `tini`, `sleep infinity`, and sshd startup

This spec does not define CLI wiring or Docker wrapper internals.

## 2. Build Contract

Both explicit builds (`havn build`) and missing-image recovery during startup
must use the same build contract:

- **Context path:** `docker/`
- **Dockerfile path:** `Dockerfile` relative to that context
- **Image tag:** the resolved havn image name (default: `havn-base:latest`)
- **Build args:** `UID` and `GID`

Equivalent Docker invocation:

```bash
docker build -t <image> --build-arg UID=<uid> --build-arg GID=<gid> docker/
```

`havn` should always detect the host user's UID/GID and pass both build args.
The Dockerfile may define `1000:1000` defaults so manual builds still work,
but the havn-managed path should not rely on those defaults.

## 3. Build Context Layout

The repository build context lives under `docker/`.

Required initial contents:

```text
docker/
  Dockerfile
```

If helper assets are added later (for example sshd config snippets), they live
in `docker/` and are copied explicitly by the Dockerfile. The build must not
depend on unrelated repository files outside the `docker/` context.

## 4. Image Contract

The base image must provide the minimum runtime needed by havn startup:

- Ubuntu 24.04 as the base OS
- Nix installed and usable with the shared `/nix` volume model
- Nix configured with `experimental-features = nix-command flakes` in
  `/etc/nix/nix.conf` so flake workflows work without per-command overrides
- a `devuser` account whose UID/GID match the detected host user
- `bash`
- `tini`
- `sudo`
- OpenSSH server with `/usr/sbin/sshd` available
- `sleep` available for the long-running container process

The image-level `/etc/nix/nix.conf` is baseline runtime config. User registry
alias persistence is a havn runtime concern and must be directed to the mounted
state path rather than persisted by mutating image-global config.

The image intentionally does not include language toolchains, editors, or
project-specific tools. Those come from `nix develop` at attach time.

## 5. Filesystem and User Layout

The image must provide these paths before runtime:

- `/home/devuser`
- `/home/devuser/.ssh`
- `/run/sshd`
- `/nix`

`devuser` owns its home directory and `.ssh` directory. The image should also
provide the XDG base directories used by havn volumes:

- `/home/devuser/.local/share`
- `/home/devuser/.cache`
- `/home/devuser/.local/state`

These may be empty in the image because named volumes are mounted over them at
runtime.

## 6. UID/GID Mapping

The `devuser` account is created or adjusted during image build using the
`UID` and `GID` build args.

Rules:

- `UID` sets the numeric UID for `devuser`
- `GID` sets the numeric primary GID for `devuser`
- the image build is responsible for ensuring the matching user/group exist
- runtime startup must not perform ad hoc UID/GID mutation inside the container

This keeps ownership stable for bind-mounted project files and host config
mounts.

## 7. Entrypoint and Init Contract

The image does not own project startup behavior. havn does.

- havn creates the container with `tini -- sleep infinity`
- root startup (`havn [path]`) attaches with interactive `nix develop <ref>#<shell>`
- plain entry (`havn enter [path]`) attaches with `bash` without automatic
  `nix develop`
- havn performs post-start init by running `sudo /usr/sbin/sshd`

There is no required custom image entrypoint script for the initial
implementation.

The image must make that post-start init viable:

- `devuser` can run `sudo /usr/sbin/sshd` without interactive password prompts
- sshd can start without additional directory creation or first-boot setup

If an implementation uses a helper script internally, it must preserve the same
external contract: havn still assumes `tini`, `sleep`, `sudo`, and
`/usr/sbin/sshd` are available.

## 8. SSHD Contract

The image must be ready for the SSH model defined in `havn-overview.md`:

- sshd listens on container port `22`
- SSH exposure to the host is controlled by havn at container creation time,
  not by the image
- public-key auth is supported for `devuser`
- password auth is disabled
- the mounted host `authorized_keys` file at
  `/home/devuser/.ssh/authorized_keys` is honored by sshd

This keeps the image aligned with `--port` publishing SSH to the host only when
the user asks for it.

## 9. Runtime Assumptions for Implementation

Startup and diagnostics may rely on these being present in the image:

- `tini`
- `sleep`
- `bash`
- `sudo`
- `/usr/sbin/sshd`

Those assumptions are part of the base-image contract and should not be
re-decided in container startup code.

For Nix registry alias persistence, runtime wiring must support a
per-container-user registry file under `/home/devuser/.local/state/nix/` so
`nix registry add` persists through the shared state volume.
