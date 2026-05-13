package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Nick-2455/marrow/internal/config"
	"github.com/Nick-2455/marrow/internal/domain"
)

func TestLoader_LoadNotFound(t *testing.T) {
	loader := config.NewLoader("/nonexistent/path/config.yaml")
	_, err := loader.Load()
	if err == nil {
		t.Fatal("expected error for missing config")
	}
	if err != domain.ErrConfigNotFound {
		t.Fatalf("expected ErrConfigNotFound, got %v", err)
	}
}

func TestLoader_LoadSaveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	loader := config.NewLoader(path)

	original := domain.Config{
		Profile:   "test-profile",
		EngramAPI: "http://test:9090",
		EngramKey: "secret-key",
	}
	original.ModelPrefs.Triage = "gpt-4"
	original.ModelPrefs.Summary = "gpt-3.5"

	if err := loader.Save(original); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := loader.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if loaded.Profile != original.Profile {
		t.Errorf("profile: got %q, want %q", loaded.Profile, original.Profile)
	}
	if loaded.EngramAPI != original.EngramAPI {
		t.Errorf("engram_api: got %q, want %q", loaded.EngramAPI, original.EngramAPI)
	}
	if loaded.EngramKey != original.EngramKey {
		t.Errorf("engram_key: got %q, want %q", loaded.EngramKey, original.EngramKey)
	}
	if loaded.ModelPrefs.Triage != original.ModelPrefs.Triage {
		t.Errorf("triage model: got %q, want %q", loaded.ModelPrefs.Triage, original.ModelPrefs.Triage)
	}
}

func TestLoader_DefaultProfile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	loader := config.NewLoader(path)

	// Write minimal config without profile
	data := []byte("engram_api_url: http://test:8080\n")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := loader.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.Profile != "default" {
		t.Errorf("expected default profile, got %q", cfg.Profile)
	}
}

func TestLoader_Path(t *testing.T) {
	path := "/some/path/config.yaml"
	loader := config.NewLoader(path)
	if loader.Path() != path {
		t.Errorf("path: got %q, want %q", loader.Path(), path)
	}
}

func TestDefaultConfigPath(t *testing.T) {
	path := config.DefaultConfigPath()
	if path == "" {
		t.Error("expected non-empty config path")
	}
}

func TestDefaultDBPath(t *testing.T) {
	path := config.DefaultDBPath()
	if path == "" {
		t.Error("expected non-empty DB path")
	}
}

func TestXDGConfigHome(t *testing.T) {
	orig := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", orig)

	os.Setenv("XDG_CONFIG_HOME", "/custom/xdg/config")
	dir := config.ConfigDir()
	expected := "/custom/xdg/config/marrow"
	if dir != expected {
		t.Errorf("config dir: got %q, want %q", dir, expected)
	}
}

func TestXDGDataHome(t *testing.T) {
	orig := os.Getenv("XDG_DATA_HOME")
	defer os.Setenv("XDG_DATA_HOME", orig)

	os.Setenv("XDG_DATA_HOME", "/custom/xdg/data")
	dir := config.DataDir()
	expected := "/custom/xdg/data/marrow"
	if dir != expected {
		t.Errorf("data dir: got %q, want %q", dir, expected)
	}
}
