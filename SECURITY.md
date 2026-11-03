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
- **Lab containers** run with `NetworkMode=none`, all capabilities dropped
  except a small allowlist, `no-new-privileges`, and memory, CPU and PID
  limits (`MEM_LIMIT`, `CPU_NANO`, `PIDS_LIMIT`, `MAX_SESSIONS_TOTAL`).
  Students are root *inside* their container, and with the default runtime
  (runc) that root talks directly to the **host kernel**: a kernel bug
  reachable from the container is a host compromise. See
  [Lab sandbox: recommended production setup](#lab-sandbox-recommended-production-setup).
- **Lab resource guards:** each user may start at most
  `SESSION_RATE_LIMIT_PER_MIN` labs per minute (default 6); labs with no
  terminal attached for `LAB_IDLE_TIMEOUT` (default 15m) are stopped so closed
  tabs do not hold slots; a lab whose container writable layer exceeds
  `LAB_DISK_MAX_BYTES` (default 2 GiB, checked every minute) is stopped and
  logged as `WARN session_disk_quota`. Where the storage driver supports it
  (overlay2 on xfs with `pquota`), `LAB_STORAGE_SIZE` (e.g. `2G`) adds a hard
  per-container limit; keep enough free disk for `MAX_SESSIONS_TOTAL` labs
  writing up to one minute past the soft limit. Verify the hard limit after
  enabling it (`docker run --rm --storage-opt size=10M alpine sh -c 'head -c 50000000 /dev/zero > /x'`
  must fail): some daemons, e.g. the containerd image store used by Docker
  Desktop, accept the option and silently ignore it.
- **Check results are best-effort anti-cheat.** Checks and setup commands run
  as `env -i PATH=... /bin/bash --noprofile --norc -c <cmd>`, so planting
  `exit 0` in `~/.bash_profile`, `/etc/profile`, `BASH_ENV` or similar no
  longer fakes a pass (covered by `TestIntegrationStartupFilesCannotFakeChecks`).
  The student is still root in the container and can replace the binaries a
  check calls (`grep`, `nginx`, `ufw`, ...). Do not use check results for
  anything with real stakes (grades, certificates) without additional review.
- **Secrets** live only in `.env` (git-ignored). `make secrets` generates random
  values; compose refuses to start without them, and the backend refuses weak
  values when `ENVIRONMENT=production`.
- **Network exposure:** only the nginx frontend port (`HTTP_PORT`, default
  8088) is published. PostgreSQL, Redis, the backend and the Docker proxy are
  on internal networks without host ports.
- **TLS:** the stack serves plain HTTP. In production put a TLS-terminating
  reverse proxy in front, set `COOKIE_SECURE=true`, and set `CORS_ORIGINS` to
  your exact `https://` origin. `COOKIE_SECURE=false` is only for
  `http://localhost`. Set `NGINX_REAL_IP_FROM` to the proxy's address as seen
  from the frontend container (one IP/CIDR) so nginx takes the client IP from
  `X-Forwarded-For`; otherwise every user shares the proxy's IP and the per-IP
  rate limits throttle everyone together. Bind `HTTP_PORT` to `127.0.0.1` so
  the proxy cannot be bypassed.
- **Admin accounts:** the backend refuses to start with
  `ENVIRONMENT=production` while any admin still uses a well-known default
  password (`admin123`, `admin`, `password`, `changeme`, `devopslab`); in
  development it logs a warning. Changing a password signs out every other
  device, including open terminals.
- **Containers** run with read-only root filesystems, `no-new-privileges`, all
  Linux capabilities dropped and memory/CPU limits. All of them run as
  non-root users except the socket proxy, which keeps uid 0 (without
  capabilities) to open the root-owned Docker socket.

## Lab sandbox: recommended production setup

Tasks may request extra capabilities in `task.yaml`; only `NET_ADMIN` is
allowlisted (the `firewall-config` task needs it for iptables/ufw). It only
affects the container's own network namespace, but with runc it exposes the
host kernel's netfilter code, historically a rich source of privilege
escalation bugs. Therefore `LAB_ALLOW_NET_ADMIN` defaults to `false` in
production unless `LAB_RUNTIME` is set; such tasks are then shown as
unavailable (`available: false`) and cannot be started.

Recommended, in order of preference:

1. **gVisor (`runsc`)**: lab syscalls are served by a user-space kernel, so
   NET_ADMIN and kernel bugs no longer reach the host kernel directly.

   ```bash
   # Debian/Ubuntu, see https://gvisor.dev/docs/user_guide/install/
   curl -fsSL https://gvisor.dev/archive.key | sudo gpg --dearmor -o /usr/share/keyrings/gvisor-archive-keyring.gpg
   echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/gvisor-archive-keyring.gpg] https://storage.googleapis.com/gvisor/releases release main" \
     | sudo tee /etc/apt/sources.list.d/gvisor.list
   sudo apt-get update && sudo apt-get install -y runsc
   sudo runsc install          # adds "runsc" to /etc/docker/daemon.json
   sudo systemctl restart docker
   docker run --rm --runtime=runsc hello-world   # smoke test
   ```

   `runsc install` writes the equivalent of:

   ```json
   { "runtimes": { "runsc": { "path": "/usr/bin/runsc" } } }
   ```

   Then set `LAB_RUNTIME=runsc` in `.env` (NET_ADMIN tasks become available
   automatically) and recreate the backend. Run
   `LAB_INTEGRATION=1 LAB_IMAGE=devopslab-lab:latest go test ./internal/session/... -run Integration`
   on the host after enabling it; some tools (systemd-analyze, ufw) behave
   differently under gVisor.
2. **User namespace remapping** (`"userns-remap": "default"` in
   `/etc/docker/daemon.json`): container root maps to an unprivileged host
   uid. It applies to the whole daemon, so the Docker socket proxy and the
   other stack containers must keep working under it; test before enabling.
   It does not reduce the kernel attack surface, so keep
   `LAB_ALLOW_NET_ADMIN=false` with it.
3. Whatever you choose, **keep the host kernel patched** (unattended security
   upgrades plus reboots) and run nothing else on the lab host.

Only set `LAB_ALLOW_NET_ADMIN=true` without a sandboxed runtime on a
throwaway host you would not mind rebuilding.

## Automated checks

CI runs gitleaks (secret scanning), govulncheck, `npm audit`, Trivy
misconfiguration scanning of Dockerfiles, and Dependabot keeps Go, npm, Docker
and GitHub Actions dependencies up to date. Every GitHub Action is pinned to a
full commit SHA (with the release tag as a comment) so a moved or compromised
tag cannot change what CI runs; Dependabot updates the pins.
