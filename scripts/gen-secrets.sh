#!/usr/bin/env bash
# Create a .env for docker compose from .env.example with random secrets.
#
# Usage: scripts/gen-secrets.sh [output-file]   (default: <repo>/.env)
#
# - Fills JWT_SECRET, DB_PASSWORD, REDIS_PASSWORD and ADMIN_PASSWORD with
#   values from `openssl rand` (hex, so they are safe in URLs/DSNs).
# - Refuses to overwrite an existing file, so real secrets are never lost.
# - The resulting file is created with mode 0600.
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
template="${repo_root}/.env.example"
target="${1:-${repo_root}/.env}"

if [[ -e "${target}" ]]; then
  echo "error: ${target} already exists; refusing to overwrite it." >&2
  echo "       Delete it first if you really want new secrets (existing data volumes keep the old DB/Redis passwords)." >&2
  exit 1
fi
if [[ ! -f "${template}" ]]; then
  echo "error: template ${template} not found." >&2
  exit 1
fi
command -v openssl >/dev/null 2>&1 || { echo "error: openssl is required." >&2; exit 1; }

jwt_secret="$(openssl rand -hex 32)"
db_password="$(openssl rand -hex 24)"
redis_password="$(openssl rand -hex 24)"
admin_password="$(openssl rand -hex 12)"

umask 077
tmp="$(mktemp "${target}.XXXXXX")"
trap 'rm -f "${tmp}"' EXIT

awk \
  -v jwt="${jwt_secret}" \
  -v db="${db_password}" \
  -v redis="${redis_password}" \
  -v admin="${admin_password}" '
  /^JWT_SECRET=/     { print "JWT_SECRET=" jwt;         next }
  /^DB_PASSWORD=/    { print "DB_PASSWORD=" db;         next }
  /^REDIS_PASSWORD=/ { print "REDIS_PASSWORD=" redis;   next }
  /^ADMIN_PASSWORD=/ { print "ADMIN_PASSWORD=" admin;   next }
  { print }
' "${template}" > "${tmp}"

for key in JWT_SECRET DB_PASSWORD REDIS_PASSWORD ADMIN_PASSWORD; do
  if ! grep -Eq "^${key}=.+" "${tmp}"; then
    echo "error: ${key} missing from ${template}; not writing ${target}." >&2
    exit 1
  fi
done

mv "${tmp}" "${target}"
trap - EXIT

echo "Created ${target} with fresh random secrets (mode 0600)."
echo "Admin login: user from ADMIN_USERNAME, password from ADMIN_PASSWORD in that file."
