package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Nick-2455/marrow/internal/domain"
)

const cacheTTL = 5 * time.Minute

// CacheSearch stores search results with a 5-minute TTL.
func (s *Store) CacheSearch(_ context.Context, query string, results []domain.Resource) error {
	data, err := json.Marshal(results)
	if err != nil {
		return fmt.Errorf("store: marshal cache results: %w", err)
	}

	now := time.Now()
	expiresAt := now.Add(cacheTTL)

	_, err = s.db.Exec(
		`INSERT INTO search_cache (query, results, cached_at, expires_at)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(query) DO UPDATE SET results = ?, cached_at = ?, expires_at = ?`,
		query, string(data), now.Unix(), expiresAt.Unix(),
		string(data), now.Unix(), expiresAt.Unix(),
	)
	if err != nil {
		return fmt.Errorf("store: cache search: %w", err)
	}

	return nil
}

// GetCachedSearch retrieves a cached search result.
// Returns (results, true, nil) on cache hit, (nil, false, nil) on miss.
func (s *Store) GetCachedSearch(_ context.Context, query string) ([]domain.Resource, bool, error) {
	var resultsJSON string
	var expiresAt int64

	err := s.db.QueryRow(
		`SELECT results, expires_at FROM search_cache WHERE query = ?`,
		query,
	).Scan(&resultsJSON, &expiresAt)

	if err == sql.ErrNoRows {
		return nil, false, nil // cache miss
	}
	if err != nil {
		return nil, false, fmt.Errorf("store: get cached search: %w", err)
	}

	// Check TTL
	if time.Now().Unix() > expiresAt {
		// Expired — delete and treat as miss
		_, _ = s.db.Exec(`DELETE FROM search_cache WHERE query = ?`, query)
		return nil, false, nil
	}

	var resources []domain.Resource
	if err := json.Unmarshal([]byte(resultsJSON), &resources); err != nil {
		return nil, false, fmt.Errorf("store: unmarshal cached results: %w", err)
	}

	return resources, true, nil
}

// InvalidateSearchCache clears all cached search results.
func (s *Store) InvalidateSearchCache(_ context.Context) error {
	_, err := s.db.Exec(`DELETE FROM search_cache`)
	if err != nil {
		return fmt.Errorf("store: invalidate search cache: %w", err)
	}
	return nil
}

// EvictExpiredCache removes all expired cache entries.
func (s *Store) EvictExpiredCache() error {
	now := time.Now().Unix()
	_, err := s.db.Exec(
		`DELETE FROM search_cache WHERE expires_at < ?`, now,
	)
	if err != nil {
		return fmt.Errorf("store: evict expired cache: %w", err)
	}
	return nil
}
