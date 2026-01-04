# duplicaCI

[![Go Version](https://img.shields.io/badge/Go-1.19+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Coverage](https://img.shields.io/badge/Coverage-96.6%25-brightgreen)](https://github.com/lioreshai/duplicaci)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

CI/CD backup orchestration for [Duplicacy Web](https://duplicacy.com/).

## Overview

**duplicaCI** complements Duplicacy Web by enabling CI/CD-controlled backup scheduling while preserving full Web UI functionality for restores, monitoring, and configuration.

```
CI Runner → SSH → Docker Host → Duplicacy Web Container
```

**Features:**
- Declarative YAML configuration
- Works with GitHub Actions, Forgejo Actions, GitLab CI, or cron
- Failure notifications via issue creation
- Automatic Web UI stats updates

## Quick Start

**1. Create config file:**

```yaml
# duplicaci.yaml
connection:
  host: root@192.168.1.100
  container: Duplicacy

backups:
  - name: appdata
    path: /mnt/appdata
    destinations: [LocalNAS, S3Backup]
    threads: 4

storages:
  LocalNAS:
    retention: { daily: 7, weekly: 4 }
  S3Backup:
    retention: { daily: 7, weekly: 4 }

notifications:
  forgejo:
    url: https://git.example.com
    repo: user/infra
```

**2. Run:**

```bash
export SSH_PASSWORD="..."
export DUPLICACY_PASSWORD="..."
./duplicaci run --config duplicaci.yaml
```

This executes: **backup → prune → check → update stats**

## CI/CD Integration

```yaml
name: Daily Backup
on:
  schedule: [{ cron: '0 6 * * *' }]
  workflow_dispatch:

jobs:
  backup:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: sudo apt-get install -y sshpass
      - run: |
          curl -sL -o duplicaci \
            "https://github.com/lioreshai/duplicaci/releases/latest/download/duplicaci_linux_amd64"
          chmod +x duplicaci
      - run: ./duplicaci run --config duplicaci.yaml
        env:
          SSH_PASSWORD: ${{ secrets.SSH_PASSWORD }}
          DUPLICACY_PASSWORD: ${{ secrets.STORAGE_PASSWORD }}
          FORGEJO_TOKEN: ${{ secrets.FORGEJO_TOKEN }}
```

## Storage Compatibility

duplicaCI works with all [Duplicacy storage backends](https://github.com/gilbertchen/duplicacy/wiki/Storage-Backends). Storage credentials (API keys, etc.) are configured in Duplicacy Web UI and stored in the container. duplicaCI only provides the encryption password at runtime.

| Storage Type | Status | Notes |
|--------------|--------|-------|
| Local disk | Full | Credentials in Web UI config |
| SFTP | Full | Credentials in Web UI config |
| Amazon S3 | Full | Also: Wasabi, MinIO, DigitalOcean Spaces |
| Backblaze B2 | Full | Credentials in Web UI config |
| Google Cloud Storage | Full | Credentials in Web UI config |
| Microsoft Azure | Full | Credentials in Web UI config |
| WebDAV | Full | Also: pCloud, Box.com |
| Google Drive | Full | OAuth token path via `gcd_token` config |
| Microsoft OneDrive | Partial | OAuth token path config not yet implemented |
| Dropbox | Partial | OAuth token path config not yet implemented |
| Hubic | Partial | OAuth token path config not yet implemented |

**Full support**: All operations work (backup, prune, check, stats).

**Partial support**: Operations work if OAuth tokens are valid, but duplicaCI cannot pass token paths to the CLI. Tokens may need periodic refresh via Web UI.

## CI/CD Compatibility

duplicaCI is a standalone binary that can be triggered by any CI/CD system or scheduler.

### Runners

Any system that can execute shell commands works:

| Platform | Status | Notes |
|----------|--------|-------|
| GitHub Actions | Full | See example workflow above |
| Forgejo Actions | Full | GitHub Actions compatible |
| GitLab CI | Full | Use shell executor |
| Jenkins | Full | Freestyle or Pipeline |
| Cron | Full | Direct execution |

### Failure Notifications

On backup failure, duplicaCI can create/update issues to alert operators.

| Platform | Status | Notes |
|----------|--------|-------|
| Forgejo | Full | Native support via `notifications.forgejo` |
| GitHub | Untested | Should work (same API as Forgejo) |
| Gitea | Untested | Should work (same API as Forgejo) |
| GitLab | Not implemented | Different API |
| Email | Not implemented | |

## Configuration

### connection

| Field | Description |
|-------|-------------|
| `host` | SSH target (user@host) |
| `container` | Docker container name |
| `gcd_token` | Google Drive token path (default: `/config/gcd-token.json`) |

### backups[]

| Field | Description |
|-------|-------------|
| `name` | Duplicacy repository ID |
| `path` | Source path to backup |
| `destinations` | Storage backends list |
| `threads` | Parallel upload threads (default: 1) |
| `cache_dir` | Duplicacy cache directory (default: uses path) |
| `retention` | Per-backup retention policy |

### storages

Storage-level retention (recommended). Pruning uses `-a` flag for efficiency.

```yaml
storages:
  MyStorage:
    retention:
      daily: 7    # keep 7 daily
      weekly: 4   # keep 4 weekly
      monthly: 3  # keep 3 monthly
```

### maintenance

Storages to prune/check but not backup to:

```yaml
maintenance:
  - LocalArray
```

### notifications.forgejo

| Field | Description |
|-------|-------------|
| `url` | Forgejo/GitHub server URL |
| `repo` | Repository for issues (owner/repo) |
| `assignee` | User to assign issues to |

## Environment Variables

| Variable | Purpose |
|----------|---------|
| `SSH_PASSWORD` | SSH password for remote host |
| `DUPLICACY_PASSWORD` | Storage encryption password |
| `FORGEJO_TOKEN` | API token for issue creation |

## Commands

```bash
# Run full workflow (recommended)
duplicaci run --config duplicaci.yaml
duplicaci run --config duplicaci.yaml --dry-run
duplicaci run --config duplicaci.yaml --verbose

# Individual operations
duplicaci backup -r myrepo --storage NAS --docker-container Duplicacy --ssh-host root@host
duplicaci prune --storage NAS --docker-container Duplicacy --ssh-host root@host
duplicaci check --storage NAS --docker-container Duplicacy --ssh-host root@host
```

## Web UI Integration

Duplicacy Web remains fully functional:
- **Restores** - Browse and restore any revision
- **Monitoring** - Dashboard stats updated by duplicaCI
- **Configuration** - Manage credentials and OAuth tokens

After migrating to CI/CD, disable scheduled jobs in the Web GUI.

## Prerequisites

- Duplicacy Web container with repositories initialized
- SSH access to Docker host
- `sshpass` installed on CI runner (`apt-get install sshpass`)

## Building

```bash
git clone https://github.com/lioreshai/duplicaci.git
cd duplicaci
go build -o duplicaci .
```

## License

MIT
