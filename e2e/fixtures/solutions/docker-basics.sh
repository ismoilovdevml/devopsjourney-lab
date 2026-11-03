#!/bin/bash
# Reference solution for tasks/docker-basics. Idempotent.
set -euo pipefail

cd /root/webapp

cat > Dockerfile <<'DOCKERFILE'
FROM nginx:1.27-alpine
COPY index.html /usr/share/nginx/html/index.html
EXPOSE 80
DOCKERFILE

cat > compose.yaml <<'COMPOSE'
services:
  web:
    build: .
    ports:
      - "8080:80"
    restart: unless-stopped
COMPOSE

hadolint --failure-threshold error Dockerfile
docker compose -f compose.yaml config -q
