# duplicaci

Run [Duplicacy](https://duplicacy.com/) backups from CI/CD pipelines.

## Use Case

You have Duplicacy Web running in a Docker container. It works great for restores and monitoring, but you want backup scheduling controlled by CI/CD where configuration is version-controlled and you get native notifications.

**duplicaci** runs Duplicacy CLI commands via SSH into your Docker container, letting you:
- Define backups declaratively in a YAML config
- Schedule via GitHub Actions, Forgejo Actions, GitLab CI, or cron
- Get failure notifications as repository issues
- Keep the Web GUI for restores and browsing

## Quick Start

### 1. Create a config file

```yaml
# duplicaci.yaml
connection:
  host: root@192.168.1.100
  container: Duplicacy

backups:
  - name: server_appdata
    path: /mnt/appdata
    destinations:
      - NASBackup
      - GoogleDrive
    retention:
      days: 14
      weeks: 180
    threads: 4

  - name: router_configs
    path: /mnt/router_backups
    destinations:
      - NASBackup
      - GoogleDrive
    retention:
      days: 14
      weeks: 180

# Optional: storages to prune/check but not backup to
maintenance:
  - LocalArray

notifications:
  forgejo:
    url: https://git.example.com
    repo: user/infra
    assignee: user
```

### 2. Run it

```bash
./duplicaci run --config duplicaci.yaml
```

This will:
1. **Backup** each source to each destination
2. **Prune** all storages (apply retention policy)
3. **Check** all storages (verify integrity)
4. **Create issue** if anything fails

## CI/CD Workflow

```yaml
name: Daily Backup

on:
  schedule:
    - cron: '0 6 * * *'
  workflow_dispatch:

jobs:
  backup:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup
        run: sudo apt-get update && sudo apt-get install -y sshpass

      - name: Download duplicaci
        run: |
          curl -sL -o duplicaci \
            "https://github.com/lioreshai/duplicaci/releases/download/v0.2.0/duplicaci_linux_amd64"
          chmod +x duplicaci

      - name: Run backups
        env:
          SSH_PASSWORD: ${{ secrets.SSH_PASSWORD }}
          DUPLICACY_PASSWORD: ${{ secrets.STORAGE_PASSWORD }}
          FORGEJO_TOKEN: ${{ secrets.FORGEJO_TOKEN }}
        run: ./duplicaci run --config duplicaci.yaml
```

That's it. No complex shell scripts, no manual orchestration.

## Configuration Reference

### connection

| Field | Description | Default |
|-------|-------------|---------|
| `host` | SSH target (user@host) | - |
| `container` | Docker container name | - |
| `gcd_token` | Google Drive token path | `/config/gcd-token.json` |

### backups[]

| Field | Description | Default |
|-------|-------------|---------|
| `name` | Duplicacy repository ID | (required) |
| `path` | Source path to backup | - |
| `cache_dir` | Duplicacy cache directory | (uses path) |
| `destinations` | Storage backends list | (required) |
| `retention.days` | Keep daily for N days | 14 |
| `retention.weeks` | Keep weekly for N days | 180 |
| `threads` | Parallel upload threads | 1 |

### maintenance

List of storage backends that need prune/check but don't receive backups.

### notifications.forgejo

| Field | Description |
|-------|-------------|
| `url` | Forgejo/GitHub server URL |
| `repo` | Repository for issues (owner/repo) |
| `assignee` | User to assign issues to |

Token is read from `FORGEJO_TOKEN` environment variable.

## Environment Variables

| Variable | Purpose |
|----------|---------|
| `SSH_PASSWORD` | SSH password for remote host |
| `DUPLICACY_PASSWORD` | Storage encryption password |
| `FORGEJO_TOKEN` | API token for issue creation |

## Commands

### run (recommended)

Run all operations from config file:

```bash
./duplicaci run --config duplicaci.yaml
./duplicaci run --config duplicaci.yaml --dry-run  # Preview only
./duplicaci run --config duplicaci.yaml --verbose  # Detailed output
```

### Individual commands

For advanced use cases, individual commands are still available:

```bash
# Backup
./duplicaci backup -r myrepo --storage NAS,GDrive \
  --docker-container Duplicacy --ssh-host root@host

# Prune
./duplicaci prune --storage NAS,GDrive \
  --docker-container Duplicacy --ssh-host root@host

# Check
./duplicaci check --storage NAS,GDrive \
  --docker-container Duplicacy --ssh-host root@host
```

## Prerequisites

1. **Duplicacy initialized** - Repositories set up via Web GUI or CLI
2. **Storage credentials** - OAuth tokens and passwords configured in Duplicacy
3. **SSH access** - Password or key-based access to backup host
4. **sshpass** - Required for password-based SSH

```bash
sudo apt-get install sshpass
```

## How It Works

duplicaci connects to your backup host via SSH, executes commands inside the Duplicacy Docker container, and handles credential injection through environment variables.

```
CI Runner → SSH → Backup Host → Docker Exec → Duplicacy CLI
```

The Web GUI remains available for:
- Restoring files
- Browsing backup history
- Monitoring storage usage
- Manual operations

Just disable scheduled jobs in the Web GUI after migrating to CI/CD.

## Building from Source

```bash
git clone https://github.com/lioreshai/duplicaci.git
cd duplicaci
go build -o duplicaci .
```

## License

MIT
