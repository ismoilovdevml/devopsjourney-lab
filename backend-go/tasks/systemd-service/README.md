# Systemd Service Creation

## Task Description
Systemd is the init system used by most modern Linux distributions to manage services. Learn how to create and manage custom systemd services.

## Objectives
1. Create a simple bash script at `/usr/local/bin/myapp.sh`
2. Create a systemd service file for the script
3. Enable the service to start on boot

## Instructions

### Step 1: Create the application script
```bash
cat > /usr/local/bin/myapp.sh << 'EOF'
#!/bin/bash
while true; do
  echo "MyApp is running..."
  sleep 60
done
EOF

chmod +x /usr/local/bin/myapp.sh
```

### Step 2: Create systemd service file
```bash
cat > /etc/systemd/system/myapp.service << 'EOF'
[Unit]
Description=My Application Service
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/myapp.sh
Restart=always

[Install]
WantedBy=multi-user.target
EOF
```

### Step 3: Enable and start the service
```bash
systemctl daemon-reload
systemctl enable myapp.service
```

## Tips
- Always reload systemd after creating/modifying service files
- Use `systemctl status myapp.service` to check service status
- Logs can be viewed with `journalctl -u myapp.service`