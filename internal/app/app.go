package app

import (
	"fmt"

	"github.com/Nick-2455/marrow/internal/config"
	"github.com/Nick-2455/marrow/internal/domain"
	"github.com/Nick-2455/marrow/internal/engram"
	"github.com/Nick-2455/marrow/internal/store"
)

// Deps holds all initialized dependencies for the application.
type Deps struct {
	Config     domain.Config
	Store      *store.Store
	Engram     domain.EngramClient
	Loader     domain.ConfigLoader
	GraphStore domain.GraphStore
}

// Bootstrap initializes all application dependencies.
//
// Order of operations:
//  1. Load configuration from disk
//  2. Open SQLite database and run migrations
//  3. Initialize Engram MCP client (stdio to engram binary)
//  4. Return assembled deps
func Bootstrap() (*Deps, error) {
	// 1. Load config
	cfgPath := config.DefaultConfigPath()
	loader := config.NewLoader(cfgPath)

	cfg, err := loader.Load()
	if err != nil {
		// ErrConfigNotFound is acceptable — use defaults
		if err != domain.ErrConfigNotFound {
			return nil, fmt.Errorf("app: load config: %w", err)
		}
		cfg, _ = loader.Load() // returns defaults
	}

	// 2. Open store and migrate
	dbPath := config.DefaultDBPath()
	s, err := store.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("app: open store: %w", err)
	}

	if err := s.Migrate(); err != nil {
		_ = s.Close()
		return nil, fmt.Errorf("app: migrate store: %w", err)
	}

	// 3. Initialize Engram MCP client
	engramClient, err := engram.NewClient(cfg.EngramPath)
	if err != nil {
		_ = s.Close()
		return nil, fmt.Errorf("app: init engram client: %w", err)
	}

	return &Deps{
		Config:     cfg,
		Store:      s,
		Engram:     engramClient,
		Loader:     loader,
		GraphStore: s, // *Store implements domain.GraphStore
	}, nil
}
