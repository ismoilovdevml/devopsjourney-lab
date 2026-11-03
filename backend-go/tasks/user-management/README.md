# Linux User & Permission Management

## Goal
Users, groups and file permissions are the first line of defence on any server.
Create an admin account and a team group, and prepare a shared directory where every
team member can deploy files.

## What you need to do
1. Create the user `devops` with home directory `/home/devops` and login shell `/bin/bash`
2. Give `devops` admin rights by adding it to the `sudo` group
3. Create the group `devteam` and add `devops` to it
4. Create `/opt/app`, owned by user `devops` and group `devteam`
5. Set the permissions of `/opt/app` to `2775`: owner and group can write, others can read,
   and the **setgid** bit makes new files inherit the `devteam` group

## Hints

### Create the user
```bash
useradd -m -s /bin/bash devops
passwd devops            # optional here
```

### Groups
```bash
usermod -aG sudo devops
groupadd devteam
usermod -aG devteam devops
id devops
```

### Shared directory
```bash
mkdir -p /opt/app
chown devops:devteam /opt/app
chmod 2775 /opt/app
ls -ld /opt/app          # drwxrwsr-x devops devteam
```

## What the checker verifies
1. user `devops` exists
2. its home is `/home/devops` (and the directory exists) and its shell is `/bin/bash`
3. `devops` is in group `sudo`
4. group `devteam` exists and `devops` is a member
5. `/opt/app` is owned by `devops:devteam`
6. `/opt/app` has mode `2775`

## Tips
- `-aG` *appends* groups; without `-a`, `usermod -G` replaces all supplementary groups
- `chmod 2775`: `2` = setgid, `7` = rwx owner, `7` = rwx group, `5` = r-x others
- `getent passwd devops` and `getent group devteam` show the account databases
