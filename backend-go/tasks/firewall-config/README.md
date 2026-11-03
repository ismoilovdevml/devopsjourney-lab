# UFW Firewall Configuration

## Goal
UFW (Uncomplicated Firewall) is the friendly front end to iptables/nftables on Ubuntu.
Apply the classic web-server policy: deny everything incoming, then open only what the
server actually serves.

> The firewall you configure here lives in the lab container's own network namespace:
> it cannot lock you out or affect anything outside the container.

## What you need to do
1. Default policies: **deny** incoming, **allow** outgoing
2. Allow SSH `22/tcp`, HTTP `80/tcp` and HTTPS `443/tcp`
3. Explicitly deny MySQL `3306/tcp`
4. Enable the firewall

## Hints

### Default policies
```bash
ufw default deny incoming
ufw default allow outgoing
```

### Rules
```bash
ufw allow 22/tcp          # always allow SSH *before* enabling a firewall on a real server
ufw allow 80/tcp
ufw allow 443/tcp
ufw deny 3306/tcp
```

### Enable and verify
```bash
ufw --force enable        # --force skips the interactive confirmation
ufw status verbose
iptables -S ufw-user-input   # the rules UFW generated for the kernel
```

## What the checker verifies
1. `ufw status` shows `Status: active`
2. default policy `deny (incoming)`
3. `22/tcp` is `ALLOW` (or `LIMIT`) from Anywhere
4. `80/tcp` is `ALLOW` from Anywhere
5. `443/tcp` is `ALLOW` from Anywhere
6. `3306/tcp` is `DENY` (or `REJECT`) from Anywhere
7. the kernel really has the rules: `iptables -S ufw-user-input` accepts 80 and 443 and drops/rejects 3306

## Common commands
```bash
ufw status numbered       # rules with numbers
ufw delete 3              # delete rule number 3
ufw limit 22/tcp          # allow SSH but rate-limit brute force attempts
ufw disable               # turn the firewall off
ufw --force reset         # start over
```
