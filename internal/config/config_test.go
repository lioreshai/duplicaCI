package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_ValidConfig(t *testing.T) {
	// Create a temporary config file
	content := `
ssh:
  host: root@192.168.1.100
  password_env: SSH_PASSWORD

docker:
  container: Duplicacy

repositories:
  - id: test_repo
    storage:
      - gdrive
      - nas
    prune: true
    prune_options: "-keep 0:30"

notifications:
  forgejo:
    url: https://git.example.com
    repo: user/repo
    token_env: FORGEJO_TOKEN
    assignee: testuser
`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Verify SSH config
	if cfg.SSH.Host != "root@192.168.1.100" {
		t.Errorf("expected SSH host 'root@192.168.1.100', got %q", cfg.SSH.Host)
	}
	if cfg.SSH.PasswordEnv != "SSH_PASSWORD" {
		t.Errorf("expected SSH password_env 'SSH_PASSWORD', got %q", cfg.SSH.PasswordEnv)
	}

	// Verify Docker config
	if cfg.Docker.Container != "Duplicacy" {
		t.Errorf("expected Docker container 'Duplicacy', got %q", cfg.Docker.Container)
	}

	// Verify repositories
	if len(cfg.Repositories) != 1 {
		t.Fatalf("expected 1 repository, got %d", len(cfg.Repositories))
	}
	repo := cfg.Repositories[0]
	if repo.ID != "test_repo" {
		t.Errorf("expected repo ID 'test_repo', got %q", repo.ID)
	}
	if len(repo.Storage) != 2 {
		t.Errorf("expected 2 storage backends, got %d", len(repo.Storage))
	}
	if !repo.Prune {
		t.Error("expected prune to be true")
	}
	if repo.PruneOptions != "-keep 0:30" {
		t.Errorf("expected prune options '-keep 0:30', got %q", repo.PruneOptions)
	}

	// Verify notifications
	if cfg.Notifications.Forgejo.URL != "https://git.example.com" {
		t.Errorf("expected Forgejo URL 'https://git.example.com', got %q", cfg.Notifications.Forgejo.URL)
	}
	if cfg.Notifications.Forgejo.Repo != "user/repo" {
		t.Errorf("expected Forgejo repo 'user/repo', got %q", cfg.Notifications.Forgejo.Repo)
	}
	if cfg.Notifications.Forgejo.Assignee != "testuser" {
		t.Errorf("expected Forgejo assignee 'testuser', got %q", cfg.Notifications.Forgejo.Assignee)
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")
	if err := os.WriteFile(configPath, []byte("not: valid: yaml: content:"), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestLoad_EmptyConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "empty.yaml")
	if err := os.WriteFile(configPath, []byte(""), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("empty config should load without error: %v", err)
	}

	// Empty config should have zero values
	if cfg.SSH.Host != "" {
		t.Errorf("expected empty SSH host, got %q", cfg.SSH.Host)
	}
	if len(cfg.Repositories) != 0 {
		t.Errorf("expected no repositories, got %d", len(cfg.Repositories))
	}
}
