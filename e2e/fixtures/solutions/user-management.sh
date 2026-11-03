#!/bin/bash
# Reference solution for tasks/user-management. Idempotent.
set -euo pipefail

id devops >/dev/null 2>&1 || useradd -m -s /bin/bash devops
getent group devteam >/dev/null || groupadd devteam
usermod -aG sudo,devteam devops

mkdir -p /opt/app
chown devops:devteam /opt/app
chmod 2775 /opt/app
ls -ld /opt/app
