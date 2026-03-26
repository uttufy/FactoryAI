// Package config handles FactoryAI configuration loading.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

// Config represents the factory configuration
type Config struct {
	Factory  FactoryConfig  `yaml:"factory"`
	Station  StationConfig  `yaml:"station"`
	Operator OperatorConfig `yaml:"operator"`
	Database DatabaseConfig `yaml:"database"`
	Beads    BeadsConfig    `yaml:"beads"`
	Logging  LoggingConfig  `yaml:"logging"`
}

// FactoryConfig contains factory-level settings
type FactoryConfig struct {
	Name        string `yaml:"name"`
	MaxStations int    `yaml:"max_stations"`
	ProjectPath string `yaml:"project_path"`
}

// StationConfig contains station settings
type StationConfig struct {
	WorktreePrefix string `yaml:"worktree_prefix"`
	DefaultBranch  string `yaml:"default_branch"`
}

// OperatorConfig contains operator settings
type OperatorConfig struct {
	HeartbeatInterval int `yaml:"heartbeat_interval"` // seconds
	StuckTimeout      int `yaml:"stuck_timeout"`      // seconds
	MaxRetries        int `yaml:"max_retries"`
}

// DatabaseConfig contains database settings
type DatabaseConfig struct {
	Path string `yaml:"path"`
}

// BeadsConfig contains beads CLI settings
type BeadsConfig struct {
	BinaryPath string `yaml:"binary_path"`
}

// LoggingConfig contains logging settings
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Factory: FactoryConfig{
			Name:        "FactoryAI",
			MaxStations: 10,
			ProjectPath: ".",
		},
		Station: StationConfig{
			WorktreePrefix: ".station-",
			DefaultBranch:  "main",
		},
		Operator: OperatorConfig{
			HeartbeatInterval: 30,
			StuckTimeout:      300, // 5 minutes
			MaxRetries:        3,
		},
		Database: DatabaseConfig{
			Path: ".factory/factory.db",
		},
		Beads: BeadsConfig{
			BinaryPath: "beads",
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "text",
		},
	}
}

// Load loads configuration from file
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	// Load .env file if exists
	_ = godotenv.Load()

	// Load YAML config if exists
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				return cfg, nil
			}
			return nil, fmt.Errorf("reading config file: %w", err)
		}

		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parsing config file: %w", err)
		}
	}

	// Override with environment variables
	if path := os.Getenv("FACTORY_DB_PATH"); path != "" {
		cfg.Database.Path = path
	}
	if path := os.Getenv("CLAUDE_BIN"); path != "" {
		// Store for later use
	}
	if path := os.Getenv("BEADS_BIN"); path != "" {
		cfg.Beads.BinaryPath = path
	}

	return cfg, nil
}

// LoadFromDir loads configuration from a directory (looks for factory.yaml)
func LoadFromDir(dir string) (*Config, error) {
	configPath := filepath.Join(dir, "factory.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return DefaultConfig(), nil
	}
	return Load(configPath)
}

// Save saves configuration to file
func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	return nil
}

// GetFactoryDir returns the factory directory path
func GetFactoryDir(projectPath string) string {
	return filepath.Join(projectPath, ".factory")
}

// EnsureFactoryDir ensures the factory directory exists
func EnsureFactoryDir(projectPath string) error {
	factoryDir := GetFactoryDir(projectPath)
	if err := os.MkdirAll(factoryDir, 0755); err != nil {
		return fmt.Errorf("creating factory directory: %w", err)
	}
	return nil
}
