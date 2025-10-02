# Cron Job & Backup Automation

## Task Description
Cron is a time-based job scheduler in Unix-like systems. Learn how to automate backups using cron jobs - a critical skill for DevOps engineers.

## Objectives
1. Create a backup script that archives `/var/www/html`
2. Make the script executable
3. Schedule it to run daily using cron

## Instructions

### Step 1: Create backup script
```bash
cat > /usr/local/bin/backup.sh << 'EOF'
#!/bin/bash
# Backup script for website files

SOURCE_DIR="/var/www/html"
BACKUP_DIR="/backups"
DATE=$(date +%Y%m%d-%H%M%S)
BACKUP_FILE="website-backup-${DATE}.tar.gz"

# Create backup directory if it doesn't exist
mkdir -p ${BACKUP_DIR}

# Create compressed backup
tar -czf ${BACKUP_DIR}/${BACKUP_FILE} ${SOURCE_DIR} 2>/dev/null

# Keep only last 7 days of backups
find ${BACKUP_DIR} -name "website-backup-*.tar.gz" -mtime +7 -delete

echo "Backup completed: ${BACKUP_FILE}"
EOF

chmod +x /usr/local/bin/backup.sh
```

### Step 2: Create backup directory
```bash
mkdir -p /backups
```

### Step 3: Test the backup script
```bash
/usr/local/bin/backup.sh
ls -lh /backups
```

### Step 4: Add cron job to run daily at 2 AM
```bash
# Add cron job
(crontab -l 2>/dev/null; echo "0 2 * * * /usr/local/bin/backup.sh") | crontab -
```

### Verify cron job
```bash
crontab -l
```

## Tips
- Cron format: `minute hour day month weekday command`
- `0 2 * * *` means: At 02:00 every day
- Use `crontab -e` to edit cron jobs interactively
- Logs are typically in `/var/log/syslog` or `/var/log/cron`
- Always test scripts before scheduling them

## Common Cron Schedule Examples
```
* * * * *     # Every minute
0 * * * *     # Every hour
0 0 * * *     # Every day at midnight
0 2 * * *     # Every day at 2 AM
0 0 * * 0     # Every Sunday at midnight
0 0 1 * *     # First day of every month
*/15 * * * *  # Every 15 minutes
```

## Testing
After setup, you can verify:
```bash
# Run backup manually
/usr/local/bin/backup.sh

# Check if backup file was created
ls -lh /backups/

# View crontab
crontab -l
```