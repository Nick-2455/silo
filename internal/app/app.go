package app

import (
	"context"
	"fmt"

	"github.com/Nick-2455/silo/internal/config"
	"github.com/Nick-2455/silo/internal/domain"
	"github.com/Nick-2455/silo/internal/engram"
	"github.com/Nick-2455/silo/internal/store"
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

	// 4. Seed demo data if graph is empty (before TUI starts — no contention)
	ctx := context.Background()
	if err := seedDemoData(ctx, deps); err != nil {
		_ = fmt.Errorf("app: seed demo data: %w", err)
	}

	// 5. Auto-create default person node if none exists
	if err := ensureDefaultPerson(ctx, deps); err != nil {
		_ = fmt.Errorf("app: ensure default person: %w", err)
	}

	return deps, nil
}

// seedDemoData populates the graph with sample data if it's empty.
func seedDemoData(ctx context.Context, deps *Deps) error {
	store := deps.GraphStore

	domains, err := store.ListNodesByType(ctx, domain.NodeTypeDomain)
	if err != nil || len(domains) > 0 {
		return err
	}

	devID := "domain/dev"
	_ = store.UpsertNode(ctx, domain.GraphNode{EngramID: devID, NodeType: domain.NodeTypeDomain, Title: "Dev", Active: true})
	_ = store.UpsertNode(ctx, domain.GraphNode{EngramID: "subarea/dev/backend", NodeType: domain.NodeTypeSubarea, Title: "Backend", Active: true})
	_ = store.AddEdge(ctx, devID, "subarea/dev/backend", domain.EdgeContains)
	_ = store.UpsertNode(ctx, domain.GraphNode{EngramID: "subarea/dev/ios", NodeType: domain.NodeTypeSubarea, Title: "iOS", Active: true})
	_ = store.AddEdge(ctx, devID, "subarea/dev/ios", domain.EdgeContains)

	filoID := "domain/filosofia"
	_ = store.UpsertNode(ctx, domain.GraphNode{EngramID: filoID, NodeType: domain.NodeTypeDomain, Title: "Filosofía", Active: true})
	_ = store.UpsertNode(ctx, domain.GraphNode{EngramID: "subarea/filosofia/estoicismo", NodeType: domain.NodeTypeSubarea, Title: "Estoicismo", Active: true})
	_ = store.AddEdge(ctx, filoID, "subarea/filosofia/estoicismo", domain.EdgeContains)

	_ = store.UpsertNode(ctx, domain.GraphNode{EngramID: "project/silo", NodeType: domain.NodeTypeProject, Title: "silo", Active: true})
	_ = store.AddEdge(ctx, "project/silo", "subarea/dev/backend", domain.EdgeAppliesTo)

	_ = store.UpsertNode(ctx, domain.GraphNode{EngramID: "project/kitting-inspection", NodeType: domain.NodeTypeProject, Title: "kitting-inspection", Active: true})
	_ = store.AddEdge(ctx, "project/kitting-inspection", "subarea/dev/backend", domain.EdgeAppliesTo)

	_ = store.UpsertNode(ctx, domain.GraphNode{EngramID: "project/publora", NodeType: domain.NodeTypeProject, Title: "publora", Active: false})
	_ = store.AddEdge(ctx, "project/publora", "subarea/dev/ios", domain.EdgeAppliesTo)

	s1ID := "session/debug-engram-client"
	_ = store.UpsertNode(ctx, domain.GraphNode{EngramID: s1ID, NodeType: domain.NodeTypeSession, Title: "Debug de Engram MCP client", Active: true})
	_ = store.AddEdge(ctx, s1ID, "project/silo", domain.EdgeWorkedOn)

	s2ID := "session/refactor-tui"
	_ = store.UpsertNode(ctx, domain.GraphNode{EngramID: s2ID, NodeType: domain.NodeTypeSession, Title: "Refactor de TUI a arquitectura Gentle AI", Active: true})
	_ = store.AddEdge(ctx, s2ID, "project/silo", domain.EdgeWorkedOn)

	l1ID := "learning/mem-update-replaces"
	_ = store.UpsertNode(ctx, domain.GraphNode{EngramID: l1ID, NodeType: domain.NodeTypeLearning, Title: "mem_update reemplaza el contenido entero — no mergea", Active: true})
	_ = store.AddEdge(ctx, l1ID, s1ID, domain.EdgeLearnedFrom)
	_ = store.AddEdge(ctx, l1ID, "subarea/dev/backend", domain.EdgeAppliesTo)
	_ = store.AddEdge(ctx, l1ID, "project/silo", domain.EdgeAppliesTo)

	l2ID := "learning/engram-json-ids"
	_ = store.UpsertNode(ctx, domain.GraphNode{EngramID: l2ID, NodeType: domain.NodeTypeLearning, Title: "Engram responde JSON con id numérico, no texto con #", Active: true})
	_ = store.AddEdge(ctx, l2ID, s1ID, domain.EdgeLearnedFrom)
	_ = store.AddEdge(ctx, l2ID, "subarea/dev/backend", domain.EdgeAppliesTo)
	_ = store.AddEdge(ctx, l2ID, "project/silo", domain.EdgeAppliesTo)

	l3ID := "learning/screen-router-pattern"
	_ = store.UpsertNode(ctx, domain.GraphNode{EngramID: l3ID, NodeType: domain.NodeTypeLearning, Title: "Patrón Screen+Router separa rendering de lógica de navegación", Active: true})
	_ = store.AddEdge(ctx, l3ID, s2ID, domain.EdgeLearnedFrom)
	_ = store.AddEdge(ctx, l3ID, "subarea/dev/backend", domain.EdgeAppliesTo)
	_ = store.AddEdge(ctx, l3ID, "subarea/dev/ios", domain.EdgeAppliesTo)
	_ = store.AddEdge(ctx, l3ID, "project/silo", domain.EdgeAppliesTo)

	return nil
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

	if deps.Engram.IsReachable(ctx) {
		_, _ = deps.Engram.SaveNode(ctx, string(domain.NodeTypePerson), "Nico", map[string]any{
			"name":   "Nico",
			"active": true,
		}, "person/default", domain.DefaultProject)
	}

	return nil
}
