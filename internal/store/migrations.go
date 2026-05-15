package store

import (
	"fmt"
)

// migrations contains the SQL statements to initialize the schema.
var migrations = []string{
	`CREATE TABLE IF NOT EXISTS triage_positions (
		resource_id TEXT PRIMARY KEY,
		bucket TEXT NOT NULL DEFAULT 'inbox',
		updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
	);`,

	`CREATE TABLE IF NOT EXISTS search_cache (
		query TEXT PRIMARY KEY,
		results TEXT NOT NULL,
		cached_at DATETIME NOT NULL DEFAULT (datetime('now')),
		expires_at DATETIME NOT NULL
	);`,

	`CREATE INDEX IF NOT EXISTS idx_triage_bucket ON triage_positions(bucket);`,
	`CREATE INDEX IF NOT EXISTS idx_cache_expires ON search_cache(expires_at);`,

	// Graph topology tables (Phase 1)
	`CREATE TABLE IF NOT EXISTS graph_nodes (
		engram_id TEXT PRIMARY KEY,
		node_type TEXT NOT NULL CHECK(node_type IN ('domain','subarea','project','session','learning','person')),
		title TEXT NOT NULL,
		active INTEGER NOT NULL DEFAULT 1,
		cached_at DATETIME NOT NULL DEFAULT (datetime('now'))
	);`,

	`CREATE TABLE IF NOT EXISTS graph_edges (
		from_id TEXT NOT NULL,
		to_id TEXT NOT NULL,
		label TEXT NOT NULL,
		created_at DATETIME NOT NULL DEFAULT (datetime('now')),
		UNIQUE(from_id, to_id, label)
	);`,

	`CREATE INDEX IF NOT EXISTS idx_edges_from ON graph_edges(from_id);`,
	`CREATE INDEX IF NOT EXISTS idx_edges_to ON graph_edges(to_id);`,
	`CREATE INDEX IF NOT EXISTS idx_graph_nodes_type ON graph_nodes(node_type);`,
	`CREATE INDEX IF NOT EXISTS idx_graph_nodes_active ON graph_nodes(node_type, active);`,
}

// Migrate runs all pending migrations. Safe to call multiple times (idempotent).
func (s *Store) Migrate() error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("store: begin migration tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for i, stmt := range migrations {
		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("store: migration %d: %w", i, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: commit migration: %w", err)
	}

	return nil
}
