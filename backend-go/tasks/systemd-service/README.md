# Systemd Service Creation

## Goal
systemd starts, supervises and restarts services on almost every modern Linux
distribution. Turn a small script into a proper service: a long-running program,
a valid unit file, and an "enabled" state so it starts at boot.

> In this lab container systemd is **not** running as PID 1, so `systemctl start`,
> `systemctl status` and `systemctl daemon-reload` report
> *"System has not been booted with systemd"*. That is expected. Offline commands work:
> `systemd-analyze verify`, `systemctl enable` and `systemctl is-enabled`. The checker
> validates your unit file with them and runs your script directly.

## What you need to do
1. Create `/usr/local/bin/myapp.sh` that
   - runs forever (a loop, it must not exit on its own),
   - appends a line containing `MyApp is running` to `/var/log/myapp.log` (once at start, then periodically),
   - is executable.
2. Create `/etc/systemd/system/myapp.service` with
   - `ExecStart=/usr/local/bin/myapp.sh`
   - `Restart=always`
   - an `[Install]` section with `WantedBy=multi-user.target`
3. Make sure `systemd-analyze verify` reports no errors.
4. Enable the service.

## Hints

### The program
```bash
cat > /usr/local/bin/myapp.sh <<'EOF2'
#!/bin/bash
while true; do
  echo "$(date -Is) MyApp is running" >> /var/log/myapp.log
  sleep 10
done
EOF2
chmod +x /usr/local/bin/myapp.sh
```

### The unit file
```ini
[Unit]
Description=My Application Service
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/myapp.sh
Restart=always

[Install]
WantedBy=multi-user.target
```

### Validate and enable
```bash
systemd-analyze verify /etc/systemd/system/myapp.service
systemctl enable myapp.service
systemctl is-enabled myapp.service
```

## What the checker verifies
1. `/usr/local/bin/myapp.sh` exists and is executable
2. `/etc/systemd/system/myapp.service` exists
3. `systemd-analyze verify` passes (it also fails if `ExecStart` is not executable)
4. `ExecStart=/usr/local/bin/myapp.sh`
5. `Restart=always`
6. `WantedBy=multi-user.target` and `systemctl is-enabled myapp.service` prints `enabled`
7. running `myapp.sh` for 3 seconds: it is still running (killed by `timeout`) and `/var/log/myapp.log` contains `MyApp is running`

## On a real server
```bash
systemctl daemon-reload
systemctl enable --now myapp.service
systemctl status myapp.service
journalctl -u myapp.service -f
```
