package app

import (
	"fmt"

	"github.com/Nick-2455/silo/internal/config"
	"github.com/Nick-2455/silo/internal/domain"
	"github.com/Nick-2455/silo/internal/engram"
	"github.com/Nick-2455/silo/internal/knowledge"
	"github.com/Nick-2455/silo/internal/store"
)

// Deps holds all initialized dependencies for the application.
type Deps struct {
	Config     domain.Config
	Store      *store.Store
	Engram     domain.EngramClient
	Loader     domain.ConfigLoader
	GraphStore domain.GraphStore
	Knowledge  *knowledge.Service
}

// Bootstrap initializes all application dependencies.
//
// Order of operations:
//  1. Load configuration from disk
//  2. Open SQLite database and run migrations
//  3. Initialize Engram MCP client (stdio to engram binary)
//  4. Build knowledge service (Engram → Obsidian bridge)
//  5. Return assembled deps
//
// The legacy graph store is left empty by default. MVP flows rely on
// Engram + Obsidian, not on pre-seeded demo data.
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

	// 3. Initialize Engram MCP client (non-fatal if unavailable)
	var engramClient domain.EngramClient
	client, err := engram.NewClient(cfg.EngramPath)
	if err != nil {
		_ = fmt.Errorf("app: init engram client (running in degraded mode): %w", err)
		engramClient = &engram.NoopClient{}
	} else {
		engramClient = client
	}

	deps := &Deps{
		Config:     cfg,
		Store:      s,
		Engram:     engramClient,
		Loader:     loader,
		GraphStore: s,
	}

	// 4. Knowledge service: wire Engram reader through the MCP client when possible.
	if mcpClient, ok := engramClient.(knowledge.MCPCaller); ok {
		deps.Knowledge = knowledge.NewService(
			knowledge.NewEngramMCPReader(mcpClient),
			knowledge.VaultStore{},
		)
	} else {
		deps.Knowledge = knowledge.NewService(nil, knowledge.VaultStore{})
	}

	return deps, nil
}
