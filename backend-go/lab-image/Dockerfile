# Lab image for DevOps Journey Lab student sessions.
#
# Every package the tasks need is baked in here so lab containers can run with
# NetworkMode=none and task setup commands finish offline in a few seconds.
#
# Build: docker build -t devopslab-lab:latest backend-go/lab-image
FROM ubuntu:24.04

ARG HADOLINT_VERSION=2.12.0
ARG HADOLINT_SHA256_AMD64=56de6d5e5ec427e17b74fa48d51271c7fc0d61244bf5c90e828aab8362d55010
ARG HADOLINT_SHA256_ARM64=5798551bf19f33951881f15eb238f90aef023f11e7ec7e9f4c37961cb87c5df6
ARG COMPOSE_VERSION=2.29.7
ARG COMPOSE_SHA256_AMD64=383ce6698cd5d5bbf958d2c8489ed75094e34a77d340404d9f32c4ae9e12baf0
ARG COMPOSE_SHA256_ARM64=6e9fbd5daa20dca5d7d89145081ae8155d68ef2928b497d9f85b54fe0f9dbb2c

ENV DEBIAN_FRONTEND=noninteractive \
    LANG=C.UTF-8 \
    TERM=xterm-256color

SHELL ["/bin/bash", "-o", "pipefail", "-c"]

# hadolint ignore=DL3008
RUN apt-get update \
 && apt-get install -y --no-install-recommends \
      bash-completion ca-certificates cron curl file gzip iproute2 iptables \
      iputils-ping jq less nano net-tools nginx procps systemd tar tree ufw vim \
 && apt-get clean \
 && rm -rf /var/lib/apt/lists/* /var/cache/apt/archives/* /var/log/apt/* /var/log/dpkg.log

# Static tooling for the docker-basics task (no Docker daemon in the lab):
# hadolint parses/lints Dockerfiles, docker-compose validates compose files.
RUN set -eux; \
    arch="$(dpkg --print-architecture)"; \
    case "$arch" in \
      amd64) h_arch=x86_64; h_sum="$HADOLINT_SHA256_AMD64"; c_arch=x86_64;  c_sum="$COMPOSE_SHA256_AMD64" ;; \
      arm64) h_arch=arm64;  h_sum="$HADOLINT_SHA256_ARM64"; c_arch=aarch64; c_sum="$COMPOSE_SHA256_ARM64" ;; \
      *) echo "unsupported architecture: $arch" >&2; exit 1 ;; \
    esac; \
    curl -fsSL --retry 5 --retry-all-errors --http1.1 -o /usr/local/bin/hadolint \
      "https://github.com/hadolint/hadolint/releases/download/v${HADOLINT_VERSION}/hadolint-Linux-${h_arch}"; \
    echo "${h_sum}  /usr/local/bin/hadolint" | sha256sum -c -; \
    curl -fsSL --retry 5 --retry-all-errors --http1.1 -o /usr/local/bin/docker-compose \
      "https://github.com/docker/compose/releases/download/v${COMPOSE_VERSION}/docker-compose-linux-${c_arch}"; \
    echo "${c_sum}  /usr/local/bin/docker-compose" | sha256sum -c -; \
    chmod 0755 /usr/local/bin/hadolint /usr/local/bin/docker-compose

COPY rootfs/ /

RUN set -eux; \
    chmod 0755 /usr/local/bin/docker; \
    # ufw tries to write /proc/sys, which is read-only in a container.
    sed -i 's|^IPT_SYSCTL=.*|IPT_SYSCTL=/etc/ufw/sysctl.lab.conf|' /etc/default/ufw; \
    : > /etc/ufw/sysctl.lab.conf; \
    # The friendly prompt + MOTD for interactive shells only.
    cat /etc/lab/bashrc.snippet >> /root/.bashrc; \
    rm -f /etc/update-motd.d/* /etc/legal; \
    # Keep the default nginx welcome page out of the way of the web task.
    rm -f /var/www/html/index.nginx-debian.html; \
    # One worker is plenty for a lab and keeps PIDs/memory low (auto = host CPUs).
    sed -i 's/^worker_processes auto;/worker_processes 1;/' /etc/nginx/nginx.conf; \
    grep -q '^worker_processes 1;' /etc/nginx/nginx.conf

WORKDIR /root
CMD ["sleep", "infinity"]
