// Package config handles YAML configuration loading, defaults, and persistence.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ListenAddress string          `yaml:"listen_address"`
	Policy        PolicyConfig    `yaml:"policy"`
	Logging       LogConfig       `yaml:"logging"`
	Intercept     InterceptConfig `yaml:"intercept"`
}

type InterceptConfig struct {
	CACert      string `yaml:"ca_cert"`
	CAKey       string `yaml:"ca_key"`
	LogBody     bool   `yaml:"log_body"`
	MaxBodySize int    `yaml:"max_body_size"`
}

type PolicyConfig struct {
	DenyFilePatterns   []string `yaml:"deny_file_patterns"`
	AllowedDirectories []string `yaml:"allowed_directories"`
}

type LogConfig struct {
	Format    string `yaml:"format"`
	File      string `yaml:"file"`
	MaxSizeMB int    `yaml:"max_size_mb"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	cfg := &Config{
		ListenAddress: "127.0.0.1:8080",
		Logging: LogConfig{
			Format: "json",
		},
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	// Apply intercept defaults
	if cfg.Intercept.CACert == "" {
		cfg.Intercept.CACert = defaultCAPath("ca.pem")
	}
	if cfg.Intercept.CAKey == "" {
		cfg.Intercept.CAKey = defaultCAPath("ca-key.pem")
	}
	if cfg.Intercept.MaxBodySize == 0 {
		cfg.Intercept.MaxBodySize = 65536
	}
	cfg.Intercept.CACert = expandHome(cfg.Intercept.CACert)
	cfg.Intercept.CAKey = expandHome(cfg.Intercept.CAKey)

	// Expand allowed directories
	for i, dir := range cfg.Policy.AllowedDirectories {
		cfg.Policy.AllowedDirectories[i] = expandHome(dir)
	}

	// Apply logging defaults
	if cfg.Logging.File == "" {
		cfg.Logging.File = defaultLogPath()
	}
	cfg.Logging.File = expandHome(cfg.Logging.File)
	if cfg.Logging.MaxSizeMB == 0 {
		cfg.Logging.MaxSizeMB = 2
	}

	return cfg, nil
}

// Save writes the config to a YAML file.
func Save(path string, cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}
	return nil
}

func defaultLogPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "audit.log"
	}
	return filepath.Join(home, ".egressor", "logs", "audit.log")
}

func defaultCAPath(filename string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filename
	}
	return filepath.Join(home, ".egressor", filename)
}

func expandHome(path string) string {
	if len(path) >= 2 && path[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}
