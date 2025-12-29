package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the duplicaci configuration file
type Config struct {
	SSH           SSHConfig           `yaml:"ssh"`
	Docker        DockerConfig        `yaml:"docker"`
	Repositories  []RepositoryConfig  `yaml:"repositories"`
	Notifications NotificationConfig  `yaml:"notifications"`
}

// SSHConfig holds SSH connection settings
type SSHConfig struct {
	Host        string `yaml:"host"`
	PasswordEnv string `yaml:"password_env"`
}

// DockerConfig holds Docker execution settings
type DockerConfig struct {
	Container string `yaml:"container"`
}

// RepositoryConfig defines a backup repository
type RepositoryConfig struct {
	ID           string   `yaml:"id"`
	Storage      []string `yaml:"storage"`
	Prune        bool     `yaml:"prune"`
	PruneOptions string   `yaml:"prune_options"`
	Check        bool     `yaml:"check"`
}

// NotificationConfig holds notification settings
type NotificationConfig struct {
	Forgejo ForgejoNotificationConfig `yaml:"forgejo"`
}

// ForgejoNotificationConfig holds Forgejo-specific notification settings
type ForgejoNotificationConfig struct {
	URL      string `yaml:"url"`
	Repo     string `yaml:"repo"`
	TokenEnv string `yaml:"token_env"`
	Assignee string `yaml:"assignee"`
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

	return &cfg, nil
}
