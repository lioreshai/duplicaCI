# duplicaci

Run [Duplicacy](https://duplicacy.com/) backups from CI/CD pipelines.

## Use Case

You have Duplicacy Web running in a Docker container. It works great for restores and monitoring, but you want backup scheduling controlled by CI/CD where configuration is version-controlled and you get native notifications.

**duplicaci** runs Duplicacy CLI commands via SSH into your Docker container, letting you:
- Schedule backups via GitHub Actions, Forgejo Actions, GitLab CI, or cron
- Keep backup configuration in git alongside your infrastructure code
- Get failure notifications as repository issues
- Use the Web GUI for restores and browsing (scheduling disabled)

## Quick Start

### 1. Download the binary

```yaml
- name: Download duplicaci
  run: |
    curl -sL -o duplicaci \
      "https://github.com/lioreshai/duplicaci/releases/download/v0.1.8/duplicaci_linux_amd64"
    chmod +x duplicaci
```

### 2. Run a backup

```bash
./duplicaci backup \
  --repository my_backup \
  --storage MyStorage \
  --docker-container Duplicacy \
  --ssh-host root@192.168.1.100
```

## Complete Workflow Example

This workflow runs daily backups to multiple storage backends with failure notifications:

```yaml
name: Daily Backup

on:
  schedule:
    - cron: '0 6 * * *'  # 6:00 UTC daily
  workflow_dispatch:
    inputs:
      dry_run:
        description: 'Dry run (print commands without executing)'
        type: boolean
        default: false

jobs:
  backup:
    runs-on: ubuntu-latest
    steps:
      - name: Setup tools
        run: sudo apt-get update && sudo apt-get install -y sshpass

      - name: Download duplicaci
        run: |
          curl -sL -o duplicaci \
            "https://github.com/lioreshai/duplicaci/releases/download/v0.1.8/duplicaci_linux_amd64"
          chmod +x duplicaci

      - name: Run backups
        env:
          SSH_PASSWORD: ${{ secrets.SSH_PASSWORD }}
          DUPLICACY_PASSWORD: ${{ secrets.STORAGE_PASSWORD }}
          FORGEJO_TOKEN: ${{ secrets.FORGEJO_TOKEN }}
        run: |
          set +e
          FAILED=0

          COMMON="--docker-container Duplicacy --ssh-host root@192.168.1.100 --verbose"
          NOTIFY="--create-issues --forgejo-url https://git.example.com --forgejo-repo user/repo --assignee user"
          STORAGES="NASBackup,GoogleDrive"

          # Phase 1: Backups
          ./duplicaci backup $COMMON $NOTIFY \
            -r server_appdata \
            --storage $STORAGES \
            --cache-dir /cache/localhost/0 \
            --gcd-token /config/gcd-token.json
          [ $? -ne 0 ] && FAILED=1

          # Phase 2: Prune old revisions
          ./duplicaci prune $COMMON \
            --storage $STORAGES \
            --cache-dir /cache/localhost/0 \
            --gcd-token /config/gcd-token.json
          [ $? -ne 0 ] && FAILED=1

          # Phase 3: Verify integrity
          ./duplicaci check $COMMON \
            --storage $STORAGES \
            --cache-dir /cache/localhost/0 \
            --gcd-token /config/gcd-token.json
          [ $? -ne 0 ] && FAILED=1

          exit $FAILED
```

## Commands

### backup

Create a backup snapshot.

```bash
./duplicaci backup \
  -r <repository_id> \
  --storage <storage1,storage2> \
  --docker-container <container_name> \
  --ssh-host <user@host> \
  --cache-dir <path> \
  --backup-options '-threads 4'
```

### prune

Remove old revisions according to retention policy.

```bash
./duplicaci prune \
  --storage <storage1,storage2> \
  --docker-container <container_name> \
  --ssh-host <user@host> \
  --cache-dir <path> \
  --prune-options '-keep 0:180 -keep 7:14 -keep 1:1 -a'
```

### check

Verify backup integrity.

```bash
./duplicaci check \
  --storage <storage1,storage2> \
  --docker-container <container_name> \
  --ssh-host <user@host> \
  --cache-dir <path>
```

## Common Options

| Flag | Description |
|------|-------------|
| `-r, --repository` | Duplicacy repository ID |
| `-s, --storage` | Storage backend(s), comma-separated |
| `--docker-container` | Docker container name to exec into |
| `--ssh-host` | SSH to host before running (user@host) |
| `--cache-dir` | Duplicacy cache directory (e.g., /cache/localhost/0) |
| `--gcd-token` | Google Drive token file path inside container |
| `--dry-run` | Print commands without executing |
| `--verbose` | Show detailed output |

### Notification Options

| Flag | Description |
|------|-------------|
| `--create-issues` | Create issue on failure |
| `--forgejo-url` | Forgejo/GitHub server URL |
| `--forgejo-repo` | Repository for issues (owner/repo) |
| `--assignee` | Assign issues to this user |

## Environment Variables

duplicaci reads credentials from environment variables:

| Variable | Purpose |
|----------|---------|
| `SSH_PASSWORD` | SSH password for remote host |
| `DUPLICACY_PASSWORD` | Storage encryption password |
| `FORGEJO_TOKEN` | API token for issue creation |

### Storage-Specific Passwords

Duplicacy uses storage-specific environment variables for non-default storages:

```
DUPLICACY_PASSWORD              # Default for all storages
DUPLICACY_MYSTORAGE_PASSWORD    # Specific to "MyStorage"
DUPLICACY_MYSTORAGE_GCD_TOKEN   # Google Drive token path for "MyStorage"
```

Storage names are uppercased with hyphens converted to underscores.

## Prerequisites

1. **Duplicacy initialized** - Repositories must be set up (via Web GUI or CLI)
2. **Storage credentials configured** - OAuth tokens and passwords in Duplicacy's keyring
3. **SSH access** - Password or key-based access to the backup host
4. **sshpass installed** - Required for password-based SSH in CI

```bash
sudo apt-get install sshpass
```

## Workflow Phases

A typical backup workflow runs three phases:

1. **Backup** - Create new snapshots for each repository
2. **Prune** - Apply retention policy to remove old revisions
3. **Check** - Verify backup integrity

Run backups first so new data is protected before any deletions.

## Failure Notifications

With `--create-issues`, duplicaci creates repository issues on failure:

- **Title**: `[duplicaci] repository_name: backup failed`
- **Body**: Error details and timestamp
- **Assignee**: Specified user (optional)

Subsequent failures on the same repository add comments to the existing issue rather than creating duplicates.

## Tips

### Continue on Error

Use `set +e` and track failures to run all backups even if one fails:

```bash
set +e
FAILED=0

./duplicaci backup ... -r repo1
[ $? -ne 0 ] && FAILED=1

./duplicaci backup ... -r repo2
[ $? -ne 0 ] && FAILED=1

exit $FAILED
```

### Dry Run

Preview commands without executing:

```bash
./duplicaci backup --dry-run --verbose ...
```

### Multiple Storages

Specify multiple storages as comma-separated values:

```bash
--storage NASBackup,GoogleDrive,S3Backup
```

### Duplicacy Web GUI

After migration, keep the Web GUI for:
- Restoring files
- Browsing backup history
- Monitoring storage usage
- Manual operations

Just disable the scheduled jobs in the Web GUI.

## Building from Source

```bash
git clone https://github.com/lioreshai/duplicaci.git
cd duplicaci
go build -o duplicaci .
```

## License

MIT
