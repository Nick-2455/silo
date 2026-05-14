package app

import (
	"context"
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

	deps := &Deps{
		Config:     cfg,
		Store:      s,
		Engram:     engramClient,
		Loader:     loader,
		GraphStore: s, // *Store implements domain.GraphStore
	}

	// 4. Auto-create default person node if none exists
	ctx := context.Background()
	if err := ensureDefaultPerson(ctx, deps); err != nil {
		// Non-fatal — log but continue
		_ = fmt.Errorf("app: ensure default person: %w", err)
	}

	return deps, nil
}

// ensureDefaultPerson creates a default person node if no person exists in the graph.
func ensureDefaultPerson(ctx context.Context, deps *Deps) error {
	// Check if any person node exists
	nodes, err := deps.GraphStore.ListNodesByType(ctx, domain.NodeTypePerson)
	if err != nil {
		return err
	}
	if len(nodes) > 0 {
		return nil // person already exists
	}

	// Create default person
	personID := "person/default"
	if err := deps.GraphStore.UpsertPerson(ctx, domain.GraphNode{
		EngramID: personID,
		NodeType: domain.NodeTypePerson,
		Title:    "Nico",
		Active:   true,
	}); err != nil {
		return fmt.Errorf("upsert person node: %w", err)
	}

	// Also save to Engram
	_, _ = deps.Engram.SaveNode(ctx, string(domain.NodeTypePerson), "Nico", map[string]any{
		"name":   "Nico",
		"active": true,
	}, "person/default")

	return nil
}
