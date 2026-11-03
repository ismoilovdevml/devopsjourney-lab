#!/bin/bash
# Reference solution for tasks/systemd-service. Idempotent.
set -euo pipefail

cat > /usr/local/bin/myapp.sh <<'SCRIPT'
#!/bin/bash
while true; do
  echo "$(date -Is) MyApp is running" >> /var/log/myapp.log
  sleep 10
done
SCRIPT
chmod 0755 /usr/local/bin/myapp.sh

cat > /etc/systemd/system/myapp.service <<'UNIT'
[Unit]
Description=My Application Service
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/myapp.sh
Restart=always

[Install]
WantedBy=multi-user.target
UNIT

systemd-analyze verify /etc/systemd/system/myapp.service
systemctl enable myapp.service
systemctl is-enabled myapp.service
