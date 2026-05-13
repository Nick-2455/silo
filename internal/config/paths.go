package config

import (
	"os"
	"path/filepath"
)

const (
	defaultEngramAPI = "http://localhost:8080"
)

// ConfigDir returns the XDG-compliant configuration directory for Marrow.
// Uses XDG_CONFIG_HOME if set, otherwise falls back to ~/.config.
func ConfigDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, appName)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory if home is unavailable
		return "."
	}
	return filepath.Join(home, ".config", appName)
}

// DataDir returns the XDG-compliant data directory for Marrow.
// Uses XDG_DATA_HOME if set, otherwise falls back to ~/.local/share.
func DataDir() string {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, appName)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(home, ".local", "share", appName)
}

// DefaultDBPath returns the default SQLite database path.
func DefaultDBPath() string {
	return filepath.Join(DataDir(), "state.db")
}

// DefaultEngramAPIURL returns the default Engram API URL.
func DefaultEngramAPIURL() string {
	return defaultEngramAPI
}
