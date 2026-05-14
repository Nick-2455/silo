package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Nick-2455/silo/internal/domain"
	"gopkg.in/yaml.v3"
)

const (
	appName      = "silo"
	configFile   = "config.yaml"
	defaultProfile = "default"
)

// Loader implements domain.ConfigLoader using YAML files on disk.
type Loader struct {
	path string
}

// NewLoader creates a ConfigLoader that reads/writes at the given path.
func NewLoader(path string) *Loader {
	return &Loader{path: path}
}

// Load reads the configuration from disk.
// If the file does not exist, returns a default config with ErrConfigNotFound.
func (l *Loader) Load() (domain.Config, error) {
	data, err := os.ReadFile(l.path)
	if err != nil {
		if os.IsNotExist(err) {
			return defaultConfig(), domain.ErrConfigNotFound
		}
		return domain.Config{}, fmt.Errorf("config: read %s: %w", l.path, err)
	}

	var cfg domain.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return domain.Config{}, fmt.Errorf("config: parse %s: %w", l.path, err)
	}

	if cfg.Profile == "" {
		cfg.Profile = defaultProfile
	}

	return cfg, nil
}

// Save writes the configuration to disk, creating parent directories as needed.
func (l *Loader) Save(cfg domain.Config) error {
	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("config: marshal: %w", err)
	}

	dir := filepath.Dir(l.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("config: mkdir %s: %w", dir, err)
	}

	if err := os.WriteFile(l.path, data, 0o644); err != nil {
		return fmt.Errorf("config: write %s: %w", l.path, err)
	}

	return nil
}

// Path returns the absolute path to the configuration file.
func (l *Loader) Path() string {
	return l.path
}

// defaultConfig returns a Config with sensible defaults.
func defaultConfig() domain.Config {
	var cfg domain.Config
	cfg.Profile = defaultProfile
	cfg.EngramPath = DefaultEngramPath()
	return cfg
}

// DefaultConfigPath returns the default configuration file path.
func DefaultConfigPath() string {
	return filepath.Join(ConfigDir(), configFile)
}
