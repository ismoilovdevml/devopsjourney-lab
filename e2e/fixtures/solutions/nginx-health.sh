#!/bin/bash
# Reference solution for tasks/nginx-health. Idempotent.
set -euo pipefail

echo ok > /var/www/html/index.html
nginx -t
pgrep -x nginx >/dev/null || service nginx start
curl -fsS http://127.0.0.1/
