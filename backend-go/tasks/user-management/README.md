# Linux User & Permission Management

## Task Description
User and permission management is crucial for system security. Learn how to create users, manage groups, and set proper file permissions.

## Objectives
1. Create a user named "devops" with sudo privileges
2. Create a directory `/opt/app` owned by devops user
3. Set correct permissions on the directory

## Instructions

### Step 1: Create user with home directory
```bash
useradd -m -s /bin/bash devops
```

### Step 2: Add user to sudo group
```bash
usermod -aG sudo devops
```

### Step 3: Set password (optional for this test)
```bash
echo "devops:password123" | chpasswd
```

### Step 4: Create directory and set ownership
```bash
mkdir -p /opt/app
chown devops:devops /opt/app
chmod 755 /opt/app
```

## Verification
```bash
# Check user details
id devops
groups devops

# Check directory permissions
ls -ld /opt/app
```

## Tips
- `-m` flag creates home directory for user
- `-s` flag sets default shell
- `chmod 755` gives rwx to owner, rx to group and others
- Use `chown user:group` to set ownership