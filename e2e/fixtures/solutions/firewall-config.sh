#!/bin/bash
# Reference solution for tasks/firewall-config. Idempotent.
set -euo pipefail

ufw default deny incoming
ufw default allow outgoing
ufw allow 22/tcp
ufw allow 80/tcp
ufw allow 443/tcp
ufw deny 3306/tcp
ufw --force enable
ufw status verbose
