# Security Policy

## Supported versions

Only the latest commit on `main` is supported. Please reproduce issues against it.

## Reporting a vulnerability

Please **do not open a public issue** for security problems.

Report privately through GitHub:
**Security → Report a vulnerability** on this repository
(GitHub private vulnerability reporting). Include:

- affected component (backend, frontend, lab image, compose/infra),
- steps to reproduce or a proof of concept,
- impact you observed.

You should get an acknowledgement within 7 days. Please allow up to 90 days
for a fix before public disclosure; we will credit you unless you prefer
otherwise.

## Threat model and deployment notes

DevOps Journey Lab gives every student a shell inside a container on the
host's Docker daemon. Treat the host as a dedicated machine for this purpose.

- **Docker API access is root-equivalent.** The backend does not mount
  `/var/run/docker.sock`; it reaches Docker through
  [docker-socket-proxy](https://github.com/Tecnativa/docker-socket-proxy) on an
  internal network, which only allows containers, exec, images, info, ping and
  version (networks, volumes, build, swarm, secrets, events, etc. are denied).
  Container *create* is still allowed, so a compromised backend could start a
  privileged container. Do not run the stack on a host shared with other
  workloads.
- **Lab containers** run with `NetworkMode=none` and memory, CPU and PID limits
  (`MEM_LIMIT`, `CPU_NANO`, `PIDS_LIMIT`, `MAX_SESSIONS_TOTAL`). Students are
  root *inside* their container; consider enabling Docker
  [user namespace remapping](https://docs.docker.com/engine/security/userns-remap/)
  or a sandboxed runtime (gVisor/Kata) on the host for stronger isolation.
- **Secrets** live only in `.env` (git-ignored). `make secrets` generates random
  values; compose refuses to start without them, and the backend refuses weak
  values when `ENVIRONMENT=production`.
- **Network exposure:** only the nginx frontend port (`HTTP_PORT`, default
  8088) is published. PostgreSQL, Redis, the backend and the Docker proxy are
  on internal networks without host ports.
- **TLS:** the stack serves plain HTTP. In production put a TLS-terminating
  reverse proxy in front, set `COOKIE_SECURE=true`, and set `CORS_ORIGINS` to
  your exact `https://` origin. `COOKIE_SECURE=false` is only for
  `http://localhost`.
- **Containers** run with read-only root filesystems, `no-new-privileges`, all
  Linux capabilities dropped and memory/CPU limits. All of them run as
  non-root users except the socket proxy, which keeps uid 0 (without
  capabilities) to open the root-owned Docker socket.

## Automated checks

CI runs gitleaks (secret scanning), govulncheck, `npm audit`, Trivy
misconfiguration scanning of Dockerfiles, and Dependabot keeps Go, npm, Docker
and GitHub Actions dependencies up to date.
