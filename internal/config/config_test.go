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

func TestRetentionConfig_ToPruneOptions(t *testing.T) {
	tests := []struct {
		name     string
		config   RetentionConfig
		expected string
	}{
		{
			name:     "default values",
			config:   RetentionConfig{},
			expected: "-keep 0:35 -keep 7:7 -keep 1:1 -a",
		},
		{
			name:     "custom daily and weekly",
			config:   RetentionConfig{Daily: 14, Weekly: 8},
			expected: "-keep 0:70 -keep 7:14 -keep 1:1 -a",
		},
		{
			name:     "with monthly",
			config:   RetentionConfig{Daily: 7, Weekly: 4, Monthly: 3},
			expected: "-keep 0:125 -keep 30:35 -keep 7:7 -keep 1:1 -a",
		},
		{
			name:     "legacy format days only",
			config:   RetentionConfig{Days: 14},
			expected: "-keep 0:180 -keep 7:14 -keep 1:1 -a",
		},
		{
			name:     "legacy format weeks only",
			config:   RetentionConfig{Weeks: 90},
			expected: "-keep 0:90 -keep 7:14 -keep 1:1 -a",
		},
		{
			name:     "legacy format both",
			config:   RetentionConfig{Days: 7, Weeks: 60},
			expected: "-keep 0:60 -keep 7:7 -keep 1:1 -a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.ToPruneOptions()
			if result != tt.expected {
				t.Errorf("ToPruneOptions() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestRetentionConfig_ToPruneOptionsWithoutAll(t *testing.T) {
	config := RetentionConfig{Daily: 7, Weekly: 4}
	result := config.ToPruneOptionsWithoutAll()
	expected := "-keep 0:35 -keep 7:7 -keep 1:1"

	if result != expected {
		t.Errorf("ToPruneOptionsWithoutAll() = %q, want %q", result, expected)
	}

	// Should not contain -a flag
	if contains(result, "-a") {
		t.Error("ToPruneOptionsWithoutAll() should not contain -a flag")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestForgejoNotificationConfig_GetToken(t *testing.T) {
	// Test direct token
	t.Run("direct token", func(t *testing.T) {
		cfg := ForgejoNotificationConfig{Token: "direct-token"}
		if got := cfg.GetToken(); got != "direct-token" {
			t.Errorf("GetToken() = %q, want %q", got, "direct-token")
		}
	})

	// Test token from custom env var
	t.Run("custom env var", func(t *testing.T) {
		os.Setenv("CUSTOM_TOKEN_VAR", "custom-env-token")
		defer os.Unsetenv("CUSTOM_TOKEN_VAR")

		cfg := ForgejoNotificationConfig{TokenEnv: "CUSTOM_TOKEN_VAR"}
		if got := cfg.GetToken(); got != "custom-env-token" {
			t.Errorf("GetToken() = %q, want %q", got, "custom-env-token")
		}
	})

	// Test default FORGEJO_TOKEN env var
	t.Run("default env var", func(t *testing.T) {
		os.Setenv("FORGEJO_TOKEN", "default-env-token")
		defer os.Unsetenv("FORGEJO_TOKEN")

		cfg := ForgejoNotificationConfig{}
		if got := cfg.GetToken(); got != "default-env-token" {
			t.Errorf("GetToken() = %q, want %q", got, "default-env-token")
		}
	})

	// Test direct token takes precedence
	t.Run("direct takes precedence", func(t *testing.T) {
		os.Setenv("FORGEJO_TOKEN", "env-token")
		defer os.Unsetenv("FORGEJO_TOKEN")

		cfg := ForgejoNotificationConfig{Token: "direct-token", TokenEnv: "FORGEJO_TOKEN"}
		if got := cfg.GetToken(); got != "direct-token" {
			t.Errorf("GetToken() = %q, want %q", got, "direct-token")
		}
	})
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name:    "no backups or repositories",
			config:  Config{},
			wantErr: true,
			errMsg:  "no backups defined",
		},
		{
			name: "backup without name",
			config: Config{
				Backups: []BackupConfig{{Destinations: []string{"storage1"}}},
			},
			wantErr: true,
			errMsg:  "name is required",
		},
		{
			name: "backup without destinations",
			config: Config{
				Backups: []BackupConfig{{Name: "test"}},
			},
			wantErr: true,
			errMsg:  "at least one destination is required",
		},
		{
			name: "valid backup config",
			config: Config{
				Backups: []BackupConfig{{Name: "test", Destinations: []string{"storage1"}}},
			},
			wantErr: false,
		},
		{
			name: "legacy repositories valid",
			config: Config{
				Repositories: []RepositoryConfig{{ID: "test"}},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				if err == nil {
					t.Error("Validate() expected error, got nil")
				} else if tt.errMsg != "" && !containsHelper(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %q, want to contain %q", err.Error(), tt.errMsg)
				}
			} else if err != nil {
				t.Errorf("Validate() unexpected error: %v", err)
			}
		})
	}
}

func TestConfig_AllStorages(t *testing.T) {
	cfg := Config{
		Backups: []BackupConfig{
			{Name: "backup1", Destinations: []string{"storage1", "storage2"}},
			{Name: "backup2", Destinations: []string{"storage2", "storage3"}},
		},
		Maintenance: []string{"storage3", "storage4"},
	}

	storages := cfg.AllStorages()

	// Should have 4 unique storages
	if len(storages) != 4 {
		t.Errorf("AllStorages() returned %d storages, want 4", len(storages))
	}

	// Check order (backup destinations first, then maintenance)
	expected := []string{"storage1", "storage2", "storage3", "storage4"}
	for i, exp := range expected {
		if i >= len(storages) || storages[i] != exp {
			t.Errorf("AllStorages()[%d] = %q, want %q", i, storages[i], exp)
		}
	}
}

func TestConfig_GetStorageRetention(t *testing.T) {
	cfg := Config{
		Storages: map[string]StorageConfig{
			"storage1": {Retention: RetentionConfig{Daily: 14, Weekly: 8}},
		},
	}

	// Existing storage
	ret, ok := cfg.GetStorageRetention("storage1")
	if !ok {
		t.Error("GetStorageRetention() should return true for existing storage")
	}
	if ret.Daily != 14 || ret.Weekly != 8 {
		t.Errorf("GetStorageRetention() = %+v, want Daily:14 Weekly:8", ret)
	}

	// Non-existing storage
	_, ok = cfg.GetStorageRetention("nonexistent")
	if ok {
		t.Error("GetStorageRetention() should return false for non-existing storage")
	}

	// Nil storages map
	cfg2 := Config{}
	_, ok = cfg2.GetStorageRetention("any")
	if ok {
		t.Error("GetStorageRetention() should return false when Storages is nil")
	}
}

func TestConfig_GetBackupRetention(t *testing.T) {
	cfg := Config{
		Backups: []BackupConfig{
			{Name: "backup1", Retention: RetentionConfig{Daily: 30, Weekly: 12}},
			{Name: "backup2", Retention: RetentionConfig{Daily: 7, Weekly: 4}},
		},
	}

	// Existing backup
	ret := cfg.GetBackupRetention("backup1")
	if ret.Daily != 30 || ret.Weekly != 12 {
		t.Errorf("GetBackupRetention('backup1') = %+v, want Daily:30 Weekly:12", ret)
	}

	// Non-existing backup returns default
	ret = cfg.GetBackupRetention("nonexistent")
	if ret.Daily != 7 || ret.Weekly != 4 {
		t.Errorf("GetBackupRetention('nonexistent') = %+v, want default Daily:7 Weekly:4", ret)
	}
}

func TestConfig_HasStorageLevelRetention(t *testing.T) {
	// With storage configs
	cfg := Config{
		Storages: map[string]StorageConfig{"storage1": {}},
	}
	if !cfg.HasStorageLevelRetention() {
		t.Error("HasStorageLevelRetention() should return true when Storages is not empty")
	}

	// Without storage configs
	cfg2 := Config{}
	if cfg2.HasStorageLevelRetention() {
		t.Error("HasStorageLevelRetention() should return false when Storages is empty")
	}
}

func TestConfig_BackupsForStorage(t *testing.T) {
	cfg := Config{
		Backups: []BackupConfig{
			{Name: "backup1", Destinations: []string{"storage1", "storage2"}},
			{Name: "backup2", Destinations: []string{"storage2", "storage3"}},
			{Name: "backup3", Destinations: []string{"storage1"}},
		},
	}

	// Storage with multiple backups
	backups := cfg.BackupsForStorage("storage1")
	if len(backups) != 2 {
		t.Errorf("BackupsForStorage('storage1') returned %d backups, want 2", len(backups))
	}

	// Storage with one backup
	backups = cfg.BackupsForStorage("storage3")
	if len(backups) != 1 || backups[0] != "backup2" {
		t.Errorf("BackupsForStorage('storage3') = %v, want [backup2]", backups)
	}

	// Non-existing storage
	backups = cfg.BackupsForStorage("nonexistent")
	if len(backups) != 0 {
		t.Errorf("BackupsForStorage('nonexistent') = %v, want empty", backups)
	}
}

func TestConfig_ApplyDefaults(t *testing.T) {
	content := `
connection:
  host: root@192.168.1.100
  container: Duplicacy

backups:
  - name: test
    destinations: [storage1]
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

	// Check defaults applied
	if cfg.Connection.GCDToken != "/config/gcd-token.json" {
		t.Errorf("GCDToken default not applied, got %q", cfg.Connection.GCDToken)
	}
	if cfg.Backups[0].Retention.Daily != 7 {
		t.Errorf("Daily default not applied, got %d", cfg.Backups[0].Retention.Daily)
	}
	if cfg.Backups[0].Retention.Weekly != 4 {
		t.Errorf("Weekly default not applied, got %d", cfg.Backups[0].Retention.Weekly)
	}
	if cfg.Backups[0].Threads != 1 {
		t.Errorf("Threads default not applied, got %d", cfg.Backups[0].Threads)
	}
}

func TestConfig_ApplyDefaults_LegacyMigration(t *testing.T) {
	content := `
ssh:
  host: root@192.168.1.100

docker:
  container: Duplicacy

backups:
  - name: test
    destinations: [storage1]
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

	// Check legacy migration
	if cfg.Connection.Host != "root@192.168.1.100" {
		t.Errorf("Legacy SSH host not migrated, got %q", cfg.Connection.Host)
	}
	if cfg.Connection.Container != "Duplicacy" {
		t.Errorf("Legacy Docker container not migrated, got %q", cfg.Connection.Container)
	}
}

func TestConfig_ApplyDefaults_LegacyRetention(t *testing.T) {
	content := `
backups:
  - name: test
    destinations: [storage1]
    retention:
      days: 14
      weeks: 90
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

	// Legacy retention should not have new defaults applied
	if cfg.Backups[0].Retention.Daily != 0 {
		t.Errorf("Daily should remain 0 for legacy config, got %d", cfg.Backups[0].Retention.Daily)
	}
	if cfg.Backups[0].Retention.Days != 14 {
		t.Errorf("Legacy Days should be preserved, got %d", cfg.Backups[0].Retention.Days)
	}
}
