#!/bin/bash
# Reference solution for tasks/cron-backup. Idempotent.
set -euo pipefail

cat > /usr/local/bin/backup.sh <<'SCRIPT'
#!/bin/bash
set -euo pipefail

SOURCE_DIR="/var/www/html"
BACKUP_DIR="/backups"
STAMP="$(date +%Y%m%d-%H%M%S)"

mkdir -p "$BACKUP_DIR"
tar -czf "$BACKUP_DIR/website-backup-$STAMP.tar.gz" "$SOURCE_DIR" 2>/dev/null

# Retention: keep 7 days of backups
find "$BACKUP_DIR" -name 'website-backup-*.tar.gz' -mtime +7 -delete

echo "Backup completed: website-backup-$STAMP.tar.gz"
SCRIPT
chmod 0755 /usr/local/bin/backup.sh

pgrep -x cron >/dev/null || service cron start

entry='0 2 * * * /usr/local/bin/backup.sh'
if ! crontab -l 2>/dev/null | grep -qxF "$entry"; then
  (crontab -l 2>/dev/null || true; echo "$entry") | crontab -
fi
crontab -l
/usr/local/bin/backup.sh
