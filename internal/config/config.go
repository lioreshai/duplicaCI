package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the duplicaci configuration file
type Config struct {
	// Connection settings
	Connection ConnectionConfig `yaml:"connection"`

	// Backup definitions
	Backups []BackupConfig `yaml:"backups"`

	// Storages that only need maintenance (prune/check), not backup
	Maintenance []string `yaml:"maintenance"`

	// Notification settings
	Notifications NotificationConfig `yaml:"notifications"`

	// Legacy fields for backward compatibility
	SSH          SSHConfig          `yaml:"ssh"`
	Docker       DockerConfig       `yaml:"docker"`
	Repositories []RepositoryConfig `yaml:"repositories"`
}

// ConnectionConfig holds connection settings
type ConnectionConfig struct {
	Host      string `yaml:"host"`      // SSH host (user@host)
	Container string `yaml:"container"` // Docker container name
	GCDToken  string `yaml:"gcd_token"` // Google Drive token path (default: /config/gcd-token.json)
}

// BackupConfig defines what to backup and where
type BackupConfig struct {
	Name         string          `yaml:"name"`         // Duplicacy repository ID
	Path         string          `yaml:"path"`         // Source path to backup
	CacheDir     string          `yaml:"cache_dir"`    // Cache directory (auto-discovered if not set)
	Destinations []string        `yaml:"destinations"` // Storage backends to backup to
	Retention    RetentionConfig `yaml:"retention"`    // Retention policy
	Threads      int             `yaml:"threads"`      // Number of backup threads (default: 1)
}

// RetentionConfig defines backup retention policy
type RetentionConfig struct {
	Days   int `yaml:"days"`   // Keep daily backups for N days (default: 14)
	Weeks  int `yaml:"weeks"`  // Keep weekly backups for N days (default: 180)
	Months int `yaml:"months"` // Keep monthly backups for N days (default: 0, disabled)
}

// ToPruneOptions converts retention config to duplicacy prune options
func (r RetentionConfig) ToPruneOptions() string {
	// Default values
	days := r.Days
	if days == 0 {
		days = 14
	}
	weeks := r.Weeks
	if weeks == 0 {
		weeks = 180
	}

	// Build prune options: -keep <n>:<d> means keep n revisions per day for d days
	// -keep 0:180 = delete all revisions older than 180 days
	// -keep 7:14 = keep weekly (every 7 days) for 14 days
	// -keep 1:1 = keep daily for 1 day
	opts := fmt.Sprintf("-keep 0:%d -keep 7:%d -keep 1:1 -a", weeks, days)

	return opts
}

// NotificationConfig holds notification settings
type NotificationConfig struct {
	Forgejo ForgejoNotificationConfig `yaml:"forgejo"`
}

// ForgejoNotificationConfig holds Forgejo-specific notification settings
type ForgejoNotificationConfig struct {
	URL      string `yaml:"url"`
	Repo     string `yaml:"repo"`
	Token    string `yaml:"token"`     // Direct token value
	TokenEnv string `yaml:"token_env"` // Environment variable name
	Assignee string `yaml:"assignee"`
}

// GetToken returns the Forgejo token, checking direct value first, then env var
func (f ForgejoNotificationConfig) GetToken() string {
	if f.Token != "" {
		return f.Token
	}
	if f.TokenEnv != "" {
		return os.Getenv(f.TokenEnv)
	}
	return os.Getenv("FORGEJO_TOKEN")
}

// Legacy types for backward compatibility
type SSHConfig struct {
	Host        string `yaml:"host"`
	PasswordEnv string `yaml:"password_env"`
}

type DockerConfig struct {
	Container string `yaml:"container"`
}

type RepositoryConfig struct {
	ID            string   `yaml:"id"`
	Path          string   `yaml:"path"`
	Storage       []string `yaml:"storage"`
	BackupOptions string   `yaml:"backup_options"`
	Prune         bool     `yaml:"prune"`
	PruneOptions  string   `yaml:"prune_options"`
	Check         bool     `yaml:"check"`
}

// Load reads and parses a config file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Apply defaults
	cfg.applyDefaults()

	return &cfg, nil
}

// applyDefaults sets default values for optional fields
func (c *Config) applyDefaults() {
	// Default GCD token path
	if c.Connection.GCDToken == "" {
		c.Connection.GCDToken = "/config/gcd-token.json"
	}

	// Apply defaults to each backup
	for i := range c.Backups {
		if c.Backups[i].Retention.Days == 0 {
			c.Backups[i].Retention.Days = 14
		}
		if c.Backups[i].Retention.Weeks == 0 {
			c.Backups[i].Retention.Weeks = 180
		}
		if c.Backups[i].Threads == 0 {
			c.Backups[i].Threads = 1
		}
	}

	// Migrate legacy config if present
	if c.Connection.Host == "" && c.SSH.Host != "" {
		c.Connection.Host = c.SSH.Host
	}
	if c.Connection.Container == "" && c.Docker.Container != "" {
		c.Connection.Container = c.Docker.Container
	}
}

// Validate checks the config for required fields
func (c *Config) Validate() error {
	if len(c.Backups) == 0 && len(c.Repositories) == 0 {
		return fmt.Errorf("no backups defined")
	}

	for i, b := range c.Backups {
		if b.Name == "" {
			return fmt.Errorf("backup[%d]: name is required", i)
		}
		if len(b.Destinations) == 0 {
			return fmt.Errorf("backup[%d] (%s): at least one destination is required", i, b.Name)
		}
	}

	return nil
}

// AllStorages returns a deduplicated list of all storage backends
func (c *Config) AllStorages() []string {
	seen := make(map[string]bool)
	var storages []string

	// Add backup destinations
	for _, b := range c.Backups {
		for _, d := range b.Destinations {
			if !seen[d] {
				seen[d] = true
				storages = append(storages, d)
			}
		}
	}

	// Add maintenance-only storages
	for _, m := range c.Maintenance {
		if !seen[m] {
			seen[m] = true
			storages = append(storages, m)
		}
	}

	return storages
}

// GetPruneOptions returns the prune options for all backups (uses first backup's retention)
func (c *Config) GetPruneOptions() string {
	if len(c.Backups) > 0 {
		return c.Backups[0].Retention.ToPruneOptions()
	}
	// Default retention
	return RetentionConfig{Days: 14, Weeks: 180}.ToPruneOptions()
}
