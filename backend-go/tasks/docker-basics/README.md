# Dockerfile & Compose Authoring

## Goal
Package a static website as a container image and describe how to run it with
Docker Compose. Writing correct Dockerfiles and compose files is the day-to-day
part of working with containers.

> This lab has **no Docker daemon** (running Docker inside a container would need a
> privileged container). You write the files, and the checker validates them statically with
> [hadolint](https://github.com/hadolint/hadolint) and `docker compose config`, the same
> tools you would use in CI. Both are installed: `hadolint` and `docker compose`.

## What you need to do
A website is waiting in `/root/webapp/index.html`. In `/root/webapp`:

1. Write a `Dockerfile` that
   - starts `FROM` the official `nginx` image **with an explicit tag** (for example `nginx:1.27-alpine`, not `latest`),
   - copies `index.html` into `/usr/share/nginx/html/`,
   - declares `EXPOSE 80`,
   - has no hadolint **errors** (warnings are fine).
2. Write a `compose.yaml` with a service named `web` that
   - is built from the Dockerfile in the same directory (`build: .`),
   - publishes host port `8080` to container port `80`.

## Hints

### Dockerfile
```dockerfile
FROM nginx:1.27-alpine
COPY index.html /usr/share/nginx/html/index.html
EXPOSE 80
```
```bash
hadolint Dockerfile
```

### compose.yaml
```yaml
services:
  web:
    build: .
    ports:
      - "8080:80"
```
```bash
docker compose config      # prints the normalized file, or an error
```

## What the checker verifies
1. `/root/webapp/Dockerfile` exists
2. `hadolint --failure-threshold error Dockerfile` passes
3. `FROM nginx:<tag>` with an explicit tag that is not `latest`
4. a `COPY` (or `ADD`) of `index.html` into `/usr/share/nginx/html`
5. `EXPOSE 80`
6. `docker compose -f compose.yaml config` accepts the file
7. service `web` has `build: .` (context `/root/webapp`)
8. service `web` maps host port `8080` to container port `80`

## On a real machine
```bash
docker compose up -d --build
curl http://localhost:8080/
```
