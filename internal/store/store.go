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
	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(15000)&_pragma=synchronous(NORMAL)")
	if err != nil {
		return nil, fmt.Errorf("store: open sqlite: %w", err)
	}

	// SQLite requires serialized access — single connection prevents SQLITE_BUSY
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	// Verify the connection works and apply pragmas
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("store: ping sqlite: %w", err)
	}

	// Ensure busy timeout is actually set (some drivers ignore DSN pragma formats).
	// This reduces SQLITE_BUSY under concurrent TUI/server usage.
	if _, err := db.Exec("PRAGMA busy_timeout = 15000"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("store: set busy_timeout: %w", err)
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
