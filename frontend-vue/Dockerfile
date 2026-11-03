# syntax=docker/dockerfile:1.7
#
# DevOps Journey Lab - frontend image (Vue SPA served by unprivileged nginx).

ARG NODE_VERSION=22
ARG NGINX_VERSION=1.29

FROM node:${NODE_VERSION}-alpine AS builder

WORKDIR /app

COPY package.json package-lock.json ./
RUN --mount=type=cache,target=/root/.npm \
    npm ci --no-audit --no-fund

COPY . .

# Same-origin API by default: nginx proxies /api and /ws to the backend.
ARG VITE_API_URL=/api
ENV VITE_API_URL=${VITE_API_URL}
RUN npm run build

FROM nginxinc/nginx-unprivileged:${NGINX_VERSION}-alpine

LABEL org.opencontainers.image.title="devopslab-frontend" \
      org.opencontainers.image.description="DevOps Journey Lab web UI" \
      org.opencontainers.image.source="https://github.com/ismoilovdevml/devopsjourney-lab" \
      org.opencontainers.image.licenses="MIT"

# nginx.conf is a template rendered by the stock entrypoint (20-envsubst-on-templates.sh)
# into /etc/nginx/conf.d/default.conf. Only NGINX_AUTH_* variables are substituted.
# With a read-only root filesystem, /etc/nginx/conf.d must be a writable tmpfs
# (see docker-compose.yml).
COPY --chown=root:root nginx.conf /etc/nginx/templates/default.conf.template
RUN rm -f /etc/nginx/conf.d/default.conf
ENV NGINX_ENVSUBST_FILTER="^NGINX_AUTH_" \
    NGINX_AUTH_RATE="10r/m" \
    NGINX_AUTH_BURST="20"
COPY --from=builder --chown=root:root /app/dist /usr/share/nginx/html

# nginx-unprivileged already runs as uid 101; be explicit.
USER 101

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD ["wget", "-q", "-O", "/dev/null", "http://127.0.0.1:8080/nginx-health"]

CMD ["nginx", "-g", "daemon off;"]
