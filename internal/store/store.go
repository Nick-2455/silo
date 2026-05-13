package store

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite" // CGO-free SQLite driver
)

// Store is the SQLite-backed implementation of domain.ResourceStore.
type Store struct {
	db *sql.DB
}

// Open opens a connection to the SQLite database at the given path.
// The database is created if it does not exist.
func Open(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("store: open sqlite: %w", err)
	}

	// Verify the connection works
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("store: ping sqlite: %w", err)
	}

	return &Store{db: db}, nil
}

// Close releases the database connection.
func (s *Store) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}

// DB returns the underlying *sql.DB for testing purposes.
func (s *Store) DB() *sql.DB {
	return s.db
}
