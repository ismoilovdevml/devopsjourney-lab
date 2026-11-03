# Cron Job & Backup Automation

## Goal
Backups that depend on someone remembering to run them do not exist. Write a backup script
with a retention policy and let cron run it every night.

## What you need to do
1. Write `/usr/local/bin/backup.sh` that
   - creates `/backups/website-backup-<timestamp>.tar.gz`, a gzip tar archive of `/var/www/html`
     (use a timestamp such as `$(date +%Y%m%d-%H%M%S)` so every run makes a new file),
   - deletes `website-backup-*.tar.gz` files in `/backups` that are **older than 7 days**,
   - exits with status `0` on success.
2. Make the script executable.
3. Start the cron daemon.
4. Add a crontab entry for root that runs the script **every day at 02:00**.

## Hints

### The script
```bash
cat > /usr/local/bin/backup.sh <<'EOF2'
#!/bin/bash
set -euo pipefail

SOURCE_DIR="/var/www/html"
BACKUP_DIR="/backups"
STAMP="$(date +%Y%m%d-%H%M%S)"

mkdir -p "$BACKUP_DIR"
tar -czf "$BACKUP_DIR/website-backup-$STAMP.tar.gz" "$SOURCE_DIR"

# Retention: keep 7 days of backups
find "$BACKUP_DIR" -name 'website-backup-*.tar.gz' -mtime +7 -delete

echo "Backup completed: website-backup-$STAMP.tar.gz"
EOF2
chmod +x /usr/local/bin/backup.sh
/usr/local/bin/backup.sh && ls -lh /backups
```

### Cron
```bash
service cron start
(crontab -l 2>/dev/null; echo "0 2 * * * /usr/local/bin/backup.sh") | crontab -
crontab -l
```

## What the checker verifies
1. `/usr/local/bin/backup.sh` exists and is executable
2. a `cron` process is running
3. root's crontab contains `0 2 * * * /usr/local/bin/backup.sh`
4. running the script creates a fresh `/backups/website-backup-*.tar.gz` that contains `index.html`
5. running the script deletes a `website-backup-*.tar.gz` file that is 10 days old

## Cron cheat sheet
```
┌──────── minute (0-59)
│ ┌────── hour (0-23)
│ │ ┌──── day of month (1-31)
│ │ │ ┌── month (1-12)
│ │ │ │ ┌ day of week (0-6, Sunday = 0)
│ │ │ │ │
0 2 * * *   every day at 02:00
*/15 * * * *  every 15 minutes
0 0 * * 0   every Sunday at midnight
```
