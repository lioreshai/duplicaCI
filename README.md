# duplicaci

**Duplicacy + CI** - A Go wrapper for running [Duplicacy](https://duplicacy.com/) backups in CI/CD pipelines.

## Overview

duplicaci is designed to orchestrate Duplicacy backup operations from CI/CD systems like GitHub Actions, Forgejo Actions, GitLab CI, or any cron-based scheduler. It provides:

- **Structured backup execution** - Run backup, prune, check, and copy operations
- **Failure notifications** - Create issues in Forgejo/GitHub when backups fail
- **Docker support** - Execute commands inside a Duplicacy container or directly via CLI
- **Dry-run mode** - Preview commands without executing
- **Exit code handling** - Proper exit codes for CI/CD pipeline integration

## Why duplicaci?

Duplicacy's Web GUI is excellent for interactive use (restores, browsing backups, monitoring), but backup scheduling is better suited for version-controlled CI/CD pipelines:

- **Version-controlled schedules** - Backup configuration lives in git
- **Unified automation** - Same platform as your other scheduled tasks
- **Built-in notifications** - Leverage CI/CD notification systems
- **Audit trail** - Complete history of backup runs in CI logs

duplicaci bridges the gap: use it for automated backups while keeping the Web GUI for everything else.

## Installation

```bash
# From source
go install github.com/lioreshai/duplicaci@latest

# Or build locally
git clone https://github.com/lioreshai/duplicaci.git
cd duplicaci
go build -o duplicaci .
```

## Quick Start

### Basic Usage

```bash
# Run a backup
duplicaci backup --repository myrepo --storage gdrive

# Run backup with prune
duplicaci backup --repository myrepo --storage gdrive --prune

# Check backup integrity
duplicaci check --repository myrepo --storage gdrive

# Dry run (show commands without executing)
duplicaci backup --repository myrepo --storage gdrive --dry-run
```

### Docker Mode

If Duplicacy runs in a Docker container (common with Duplicacy Web):

```bash
# Execute inside container
duplicaci backup \
  --repository myrepo \
  --storage gdrive \
  --docker-container Duplicacy

# With SSH to remote host
duplicaci backup \
  --repository myrepo \
  --storage gdrive \
  --docker-container Duplicacy \
  --ssh-host root@192.168.1.100
```

### CI/CD Integration

#### Forgejo/GitHub Actions

```yaml
name: Daily Backup
on:
  schedule:
    - cron: '0 1 * * *'
  workflow_dispatch:

jobs:
  backup:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.21'

      - name: Install duplicaci
        run: go install github.com/lioreshai/duplicaci@latest

      - name: Setup SSH
        run: |
          mkdir -p ~/.ssh
          echo "${{ secrets.SSH_PRIVATE_KEY }}" | base64 -d > ~/.ssh/id_rsa
          chmod 600 ~/.ssh/id_rsa

      - name: Run backup
        env:
          SSH_PASSWORD: ${{ secrets.SSH_PASSWORD }}
          FORGEJO_TOKEN: ${{ secrets.FORGEJO_TOKEN }}
        run: |
          duplicaci backup \
            --config backup-config.yaml \
            --ssh-host root@192.168.1.100 \
            --ssh-password "$SSH_PASSWORD" \
            --create-issues \
            --forgejo-url https://git.example.com \
            --forgejo-repo myuser/duplicaci
```

## Configuration

### Command Line

```bash
duplicaci backup [flags]

Flags:
      --repository string      Repository ID to backup
      --storage string         Storage backend name (can be specified multiple times)
      --prune                  Run prune after backup
      --prune-options string   Prune options (default: "-keep 0:180 -keep 7:14 -keep 1:1 -a")
      --check                  Run check after backup
      --docker-container string  Run inside Docker container
      --ssh-host string        SSH to host before running (user@host)
      --ssh-password string    SSH password (or use SSH_PASSWORD env var)
      --dry-run                Print commands without executing
      --create-issues          Create Forgejo/GitHub issue on failure
      --forgejo-url string     Forgejo server URL
      --forgejo-repo string    Repository for issues (owner/repo)
      --forgejo-token string   Forgejo API token (or FORGEJO_TOKEN env var)
      --config string          Config file path
```

### Config File

```yaml
# backup-config.yaml
ssh:
  host: root@192.168.1.100
  password_env: SSH_PASSWORD  # Read from environment variable

docker:
  container: Duplicacy

repositories:
  - id: server_appdata
    storage:
      - gdrive
      - nas-backup
    prune: true
    prune_options: "-keep 0:180 -keep 7:14 -keep 1:1 -a"

  - id: network_config
    storage:
      - gdrive
      - nas-backup
    prune: true

notifications:
  forgejo:
    url: https://git.example.com
    repo: myuser/duplicaci
    token_env: FORGEJO_TOKEN
    assignee: myuser
```

## Notifications

When `--create-issues` is enabled, duplicaci creates or updates issues on backup failure:

- **New failure**: Creates issue with error details and logs
- **Repeated failure**: Adds comment to existing open issue
- **Recovery**: Adds success comment (doesn't auto-close)

Issue title format: `[duplicaci] repository: backup failed`

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Backup failed |
| 2 | Configuration error |
| 3 | SSH/connection error |
| 4 | Notification error (backup may have succeeded) |

## Development

```bash
# Build
go build -o duplicaci .

# Test
go test ./...

# Run locally
./duplicaci backup --dry-run --repository test --storage local
```

## License

MIT License - see [LICENSE](LICENSE) for details.

## Related Projects

- [Duplicacy](https://duplicacy.com/) - The backup tool this wraps
- [duplicacy-util](https://github.com/jeffaco/duplicacy-util) - Another CLI wrapper with email notifications
