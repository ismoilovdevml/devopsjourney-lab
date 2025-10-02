# UFW Firewall Configuration

## Task Description
UFW (Uncomplicated Firewall) is a user-friendly frontend for iptables. Learn how to configure basic firewall rules to secure your server.

## Objectives
1. Configure UFW firewall rules
2. Allow SSH, HTTP, and HTTPS traffic
3. Enable the firewall

## Instructions

### Step 1: Set default policies
```bash
ufw default deny incoming
ufw default allow outgoing
```

### Step 2: Allow required services
```bash
# Allow SSH (important - don't lock yourself out!)
ufw allow 22/tcp

# Allow HTTP
ufw allow 80/tcp

# Allow HTTPS
ufw allow 443/tcp
```

### Step 3: Enable firewall
```bash
ufw --force enable
```

### Step 4: Verify configuration
```bash
ufw status verbose
```

## Tips
- Always allow SSH before enabling firewall to prevent lockout
- Use `ufw status numbered` to see rule numbers
- Delete rules with `ufw delete [rule-number]`
- You can also use service names: `ufw allow ssh`
- `--force` flag is needed in non-interactive environments

## Common Commands
```bash
ufw status                  # Check firewall status
ufw enable                  # Enable firewall
ufw disable                 # Disable firewall
ufw allow [port]/[protocol] # Allow port
ufw deny [port]             # Deny port
ufw delete allow [port]     # Remove rule
```