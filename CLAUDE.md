# duplicaci Development Guide

## Testing

### Unit Tests

Run all unit tests:
```bash
go test -v ./...
```

Run tests with coverage:
```bash
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### Integration Tests

Integration tests require environment variables and are tagged with `integration`:

```bash
# Set up environment
export INTEGRATION_SSH_HOST=root@192.168.1.100
export INTEGRATION_SSH_PASSWORD=yourpassword
export INTEGRATION_DOCKER_CONTAINER=Duplicacy

# Run integration tests
go test -tags=integration -v ./internal/executor/
```

Integration tests verify:
- SSH connection works
- Docker exec command building
- Duplicacy list command (read-only)

### Manual Testing

```bash
# Build
go build -o duplicaci .

# Dry run (shows commands without executing)
./duplicaci backup \
  --repository test \
  --storage gdrive \
  --docker-container Duplicacy \
  --ssh-host root@192.168.1.100 \
  --dry-run \
  --verbose
```

## Building

### Local Build

```bash
go build -o duplicaci .
```

### Cross-Platform Build

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o duplicaci-linux-amd64 .

# macOS
GOOS=darwin GOARCH=amd64 go build -o duplicaci-darwin-amd64 .

# Windows
GOOS=windows GOARCH=amd64 go build -o duplicaci-windows-amd64.exe .
```

### With Version Info

```bash
VERSION=v0.1.0
COMMIT=$(git rev-parse --short HEAD)
DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)

go build -ldflags "-X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}" -o duplicaci .
```

## Releasing

Releases are automated via Forgejo Actions when you push a tag:

```bash
git tag v0.1.0
git push origin v0.1.0
```

This triggers the release workflow which:
1. Builds binaries for Linux, macOS, Windows (amd64 + arm64)
2. Creates checksums
3. Creates a GitHub/Forgejo release with the binaries

## Project Structure

```
duplicaci/
├── cmd/           # CLI commands (cobra)
│   ├── root.go    # Root command, global flags
│   ├── backup.go  # Backup subcommand
│   ├── check.go   # Check subcommand
│   └── prune.go   # Prune subcommand
├── internal/
│   ├── config/    # YAML config file parsing
│   ├── executor/  # Command execution (local, SSH, Docker)
│   └── notifier/  # Forgejo/GitHub issue notifications
├── docs/          # Implementation documentation
└── main.go        # Entry point
```

## Code Style

- Run `go fmt ./...` before committing
- Run `go vet ./...` to check for issues
- Tests are required for new functionality
