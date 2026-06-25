# Docker-compatible runtime guide

This is a derivative guide for choosing and configuring the host container
runtime endpoint used by `havn`.

## Support statement

`havn` supports Docker-compatible daemon endpoints. The current production
adapter talks to the Docker API, so supported providers include Docker Desktop,
Docker Engine, and Colima when Colima is started with its Docker runtime.

Colima support does not mean a separate Colima backend. It means Colima provides
the Docker API server that `havn` talks to.

## Colima setup

Start Colima with Docker runtime:

```bash
colima start --runtime docker --ssh-agent
```

Point `havn` at Colima's Docker socket with `DOCKER_HOST`:

```bash
export DOCKER_HOST="unix://$HOME/.colima/default/docker.sock"
```

You can also derive the socket from the Docker CLI context:

```bash
export DOCKER_HOST="$(docker context inspect colima --format '{{ (index .Endpoints "docker").Host }}')"
```

Do not assume `docker context use colima` alone affects `havn`. `havn` reads
Docker SDK environment configuration; `DOCKER_HOST` is the reliable control
surface.

## State separation when switching providers

Docker Desktop, Docker Engine, and Colima keep separate images, containers,
networks, and volumes. When switching the daemon endpoint, `havn` may recreate
missing runtime resources, but existing data is not shared automatically.

If shared Dolt data matters, back up or migrate the relevant Docker volumes before switching providers.

## Provider-specific checks

When validating Colima or another VM-backed endpoint, check these workflows
under that daemon:

1. `havn doctor`
2. `havn build`
3. `havn up .`
4. `havn enter .`
5. shared Dolt startup and database provisioning
6. published service ports
7. SSH agent forwarding

Projects must live in a host path visible to the selected daemon. With Colima,
projects under the host user's home directory usually work, but custom Colima
mount configuration can affect bind mounts. Published ports and SSH agent
sockets may also behave differently with VM-backed daemon endpoints.

For SSH agent forwarding on Colima, configure havn with Colima's
Docker-daemon-visible SSH agent socket instead of the macOS host
`SSH_AUTH_SOCK` path:

```toml
[mounts.ssh]
forward_agent = true
agent_socket = "/run/host-services/ssh-auth.sock"
```

`agent_socket` is ignored when `forward_agent=false`. When unset, havn falls
back to the host `SSH_AUTH_SOCK` value if that path is visible to the selected
Docker-compatible daemon.
