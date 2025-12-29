# Implementation Guide

This document describes how to set up duplicaci for automated backups via CI/CD.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    CI/CD Platform                           │
│  (Forgejo Actions / GitHub Actions / GitLab CI)             │
│                                                             │
│  Scheduled workflow triggers duplicaci                      │
└─────────────────────────────────────────────────────────────┘
                              │
                              │ SSH (optional)
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    Backup Host                              │
│                                                             │
│  ┌─────────────────────────┐                               │
│  │   Duplicacy Container   │  ← duplicaci executes         │
│  │   (or CLI binary)       │    commands here              │
│  │                         │                               │
│  │ • OAuth tokens          │                               │
│  │ • Storage passwords     │                               │
│  │ • Repository configs    │                               │
│  └─────────────────────────┘                               │
│               │                                             │
│               ▼                                             │
│  ┌─────────────────────────┐                               │
│  │   Backup Sources        │                               │
│  │   (mounted volumes)     │                               │
│  └─────────────────────────┘                               │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    Storage Backends                         │
│  • Google Drive                                             │
│  • Local NAS                                                │
│  • S3 / B2 / etc.                                           │
└─────────────────────────────────────────────────────────────┘
```

## Prerequisites

1. **Duplicacy initialized** - Either via Web GUI or CLI, the repositories must be initialized with `duplicacy init`
2. **Storage credentials** - OAuth tokens, passwords, etc. must be configured in the Duplicacy keyring
3. **SSH access** - If running remotely, SSH access to the backup host
4. **CI/CD secrets** - SSH keys/passwords and API tokens stored as secrets

## Setup Steps

### 1. Verify Existing Duplicacy Setup

If you have Duplicacy Web running, your repositories are already initialized:

```bash
# List repositories inside the container
docker exec Duplicacy duplicacy list

# Check storage configurations
docker exec Duplicacy cat /config/duplicacy.json
```

### 2. Create Workflow File

Create `.forgejo/workflows/backup.yaml` (or equivalent for your CI platform):

```yaml
name: Daily Backup

on:
  schedule:
    # Run daily at 1:00 AM
    - cron: '0 1 * * *'
  workflow_dispatch:  # Allow manual trigger

jobs:
  backup:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout repository
        uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.21'
          cache: false

      - name: Build duplicaci
        run: go build -o duplicaci .

      - name: Setup SSH key
        run: |
          mkdir -p ~/.ssh
          echo "${{ secrets.SSH_PRIVATE_KEY }}" | base64 -d > ~/.ssh/id_rsa
          chmod 600 ~/.ssh/id_rsa

      - name: Install sshpass
        run: sudo apt-get update && sudo apt-get install -y sshpass

      - name: Run backup - server_appdata
        env:
          SSH_PASSWORD: ${{ secrets.SSH_PASSWORD }}
          FORGEJO_TOKEN: ${{ secrets.FORGEJO_TOKEN }}
        run: |
          ./duplicaci backup \
            --repository server_appdata \
            --storage gdrive \
            --storage nas-backup \
            --prune \
            --docker-container Duplicacy \
            --ssh-host root@192.168.1.100 \
            --create-issues \
            --forgejo-url https://git.example.com \
            --forgejo-repo myuser/duplicaci \
            --assignee myuser

      - name: Run backup - network_config
        env:
          SSH_PASSWORD: ${{ secrets.SSH_PASSWORD }}
          FORGEJO_TOKEN: ${{ secrets.FORGEJO_TOKEN }}
        run: |
          ./duplicaci backup \
            --repository network_config \
            --storage gdrive \
            --storage nas-backup \
            --prune \
            --docker-container Duplicacy \
            --ssh-host root@192.168.1.100 \
            --create-issues \
            --forgejo-url https://git.example.com \
            --forgejo-repo myuser/duplicaci \
            --assignee myuser
```

### 3. Configure CI/CD Secrets

Add these secrets to your CI/CD platform:

| Secret | Description |
|--------|-------------|
| `SSH_PRIVATE_KEY` | Base64-encoded SSH private key for backup host |
| `SSH_PASSWORD` | SSH password for backup host |
| `FORGEJO_TOKEN` | Forgejo API token for issue creation |

### 4. Disable Web GUI Schedules

To avoid duplicate backups, disable the schedules in Duplicacy Web:

1. Open Duplicacy Web UI
2. Go to Schedules
3. Remove or disable backup jobs (keep prune/check if desired)

The Web GUI remains fully functional for:
- Browsing backups
- Restoring files
- Manual operations
- Viewing logs

### 5. Test the Setup

Run manually first:

```bash
# Dry run to see commands
./duplicaci backup \
  --repository server_appdata \
  --storage gdrive \
  --docker-container Duplicacy \
  --ssh-host root@192.168.1.100 \
  --dry-run

# Real run
./duplicaci backup \
  --repository server_appdata \
  --storage gdrive \
  --docker-container Duplicacy \
  --ssh-host root@192.168.1.100
```

## Troubleshooting

### SSH Connection Fails

```bash
# Test SSH manually
sshpass -p "$SSH_PASSWORD" ssh -o StrictHostKeyChecking=no root@192.168.1.100 'echo connected'

# Check if sshpass is installed
which sshpass || sudo apt-get install sshpass
```

### Docker Exec Fails

```bash
# Verify container is running
ssh root@192.168.1.100 'docker ps | grep -i duplicacy'

# Test docker exec
ssh root@192.168.1.100 'docker exec Duplicacy duplicacy list'
```

### Duplicacy Command Fails

```bash
# Check Duplicacy logs
ssh root@192.168.1.100 'docker exec Duplicacy ls -la /config/logs/'

# View recent log
ssh root@192.168.1.100 'docker exec Duplicacy tail -100 /config/logs/$(docker exec Duplicacy ls -t /config/logs/ | head -1)'
```

### Token/Credential Issues

If Google Drive or other OAuth storage fails:
1. Open Duplicacy Web UI
2. Manually trigger a backup to refresh tokens
3. Tokens are stored in `/config/gcd-token.json` (Google Drive)

## Monitoring

### CI/CD Logs

All backup output is visible in the CI/CD job logs.

### Issue Notifications

Failed backups create issues in the configured repository:
- Title: `[duplicaci] <repository>: backup failed`
- Body: List of errors encountered
- Subsequent failures add comments to existing issue

### Web GUI

The Duplicacy Web GUI shows backup history and can be used to verify backups completed successfully.

## Extending

### Adding New Repositories

1. Initialize the repository with Duplicacy (via Web GUI or CLI)
2. Add a new step to the workflow file
3. Or add to the config file if using `--config`

### Multiple Storage Backends

Specify multiple `--storage` flags:

```bash
./duplicaci backup \
  --repository myrepo \
  --storage backend1 \
  --storage backend2 \
  --storage backend3
```

### Custom Retention Policies

Override default prune options:

```bash
./duplicaci backup \
  --repository myrepo \
  --storage gdrive \
  --prune \
  --prune-options "-keep 0:365 -keep 30:30 -keep 7:7 -keep 1:1 -a"
```
