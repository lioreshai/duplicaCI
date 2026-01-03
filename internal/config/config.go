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

	// Storage configurations (retention, etc.)
	Storages map[string]StorageConfig `yaml:"storages"`

	// Storages that only need maintenance (prune/check), not backup
	Maintenance []string `yaml:"maintenance"`

	// Notification settings
	Notifications NotificationConfig `yaml:"notifications"`

	// Legacy fields for backward compatibility
	SSH          SSHConfig          `yaml:"ssh"`
	Docker       DockerConfig       `yaml:"docker"`
	Repositories []RepositoryConfig `yaml:"repositories"`
}

// StorageConfig defines per-storage settings
type StorageConfig struct {
	Retention RetentionConfig `yaml:"retention"` // Retention policy for this storage
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
	// New format: specify counts
	Daily   int `yaml:"daily"`   // Number of daily backups to keep (default: 7)
	Weekly  int `yaml:"weekly"`  // Number of weekly backups to keep (default: 4)
	Monthly int `yaml:"monthly"` // Number of monthly backups to keep (default: 0)

	// Legacy format (deprecated)
	Days  int `yaml:"days"`  // Keep daily backups for N days
	Weeks int `yaml:"weeks"` // Keep weekly backups for N days
}

// ToPruneOptions converts retention config to duplicacy prune options (with -a flag)
func (r RetentionConfig) ToPruneOptions() string {
	return r.toPruneOptions(true)
}

// ToPruneOptionsWithoutAll converts retention config to prune options without -a flag
// Used when pruning specific repositories with -id
func (r RetentionConfig) ToPruneOptionsWithoutAll() string {
	return r.toPruneOptions(false)
}

// toPruneOptions is the internal implementation
func (r RetentionConfig) toPruneOptions(includeAll bool) string {
	allFlag := ""
	if includeAll {
		allFlag = " -a"
	}

	// Handle legacy format
	if r.Days > 0 || r.Weeks > 0 {
		days := r.Days
		if days == 0 {
			days = 14
		}
		weeks := r.Weeks
		if weeks == 0 {
			weeks = 180
		}
		return fmt.Sprintf("-keep 0:%d -keep 7:%d -keep 1:1%s", weeks, days, allFlag)
	}

	// New format: counts
	daily := r.Daily
	if daily == 0 {
		daily = 7
	}
	weekly := r.Weekly
	if weekly == 0 {
		weekly = 4
	}
	monthly := r.Monthly

	// Calculate day boundaries
	// Daily: days 1 to daily
	// Weekly: days daily+1 to daily + (weekly * 7)
	// Monthly: days weekly_end+1 to weekly_end + (monthly * 30)
	dailyEnd := daily
	weeklyEnd := dailyEnd + (weekly * 7)

	var opts string
	if monthly > 0 {
		monthlyEnd := weeklyEnd + (monthly * 30)
		opts = fmt.Sprintf("-keep 0:%d -keep 30:%d -keep 7:%d -keep 1:1%s", monthlyEnd, weeklyEnd, dailyEnd, allFlag)
	} else {
		opts = fmt.Sprintf("-keep 0:%d -keep 7:%d -keep 1:1%s", weeklyEnd, dailyEnd, allFlag)
	}

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
		// Only set new format defaults if legacy format not used
		if c.Backups[i].Retention.Days == 0 && c.Backups[i].Retention.Weeks == 0 {
			if c.Backups[i].Retention.Daily == 0 {
				c.Backups[i].Retention.Daily = 7
			}
			if c.Backups[i].Retention.Weekly == 0 {
				c.Backups[i].Retention.Weekly = 4
			}
			// Monthly defaults to 0 (disabled)
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

// GetStorageRetention returns the retention config for a storage, if defined
func (c *Config) GetStorageRetention(storage string) (RetentionConfig, bool) {
	if c.Storages != nil {
		if sc, ok := c.Storages[storage]; ok {
			return sc.Retention, true
		}
	}
	return RetentionConfig{}, false
}

// GetBackupRetention returns the retention config for a specific backup
func (c *Config) GetBackupRetention(backupName string) RetentionConfig {
	for _, b := range c.Backups {
		if b.Name == backupName {
			return b.Retention
		}
	}
	// Default retention
	return RetentionConfig{Daily: 7, Weekly: 4}
}

// HasStorageLevelRetention checks if any storage has retention defined
func (c *Config) HasStorageLevelRetention() bool {
	return len(c.Storages) > 0
}

// BackupsForStorage returns all backup names that target a specific storage
func (c *Config) BackupsForStorage(storage string) []string {
	var backups []string
	for _, b := range c.Backups {
		for _, d := range b.Destinations {
			if d == storage {
				backups = append(backups, b.Name)
				break
			}
		}
	}
	return backups
}
