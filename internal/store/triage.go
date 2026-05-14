package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Nick-2455/silo/internal/domain"
)

// GetTriagePosition retrieves the triage position for a resource.
func (s *Store) GetTriagePosition(_ context.Context, resourceID string) (domain.TriagePosition, error) {
	var pos domain.TriagePosition
	var updatedAt int64

	err := s.db.QueryRow(
		`SELECT resource_id, bucket, updated_at FROM triage_positions WHERE resource_id = ?`,
		resourceID,
	).Scan(&pos.ResourceID, &pos.Bucket, &updatedAt)

	if err == sql.ErrNoRows {
		return pos, domain.ErrTriageNotFound
	}
	if err != nil {
		return pos, fmt.Errorf("store: get triage position: %w", err)
	}

	pos.UpdatedAt = time.Unix(updatedAt, 0)

	return pos, nil
}

// SetTriagePosition creates or updates a triage position.
func (s *Store) SetTriagePosition(_ context.Context, pos domain.TriagePosition) error {
	now := time.Now()
	if pos.UpdatedAt.IsZero() {
		pos.UpdatedAt = now
	}

	_, err := s.db.Exec(
		`INSERT INTO triage_positions (resource_id, bucket, updated_at)
		 VALUES (?, ?, ?)
		 ON CONFLICT(resource_id) DO UPDATE SET bucket = ?, updated_at = ?`,
		pos.ResourceID, pos.Bucket, pos.UpdatedAt.Unix(),
		pos.Bucket, pos.UpdatedAt.Unix(),
	)
	if err != nil {
		return fmt.Errorf("store: set triage position: %w", err)
	}

	return nil
}

// GetAllTriagePositions returns all triage positions.
func (s *Store) GetAllTriagePositions(_ context.Context) ([]domain.TriagePosition, error) {
	rows, err := s.db.Query(
		`SELECT resource_id, bucket, updated_at FROM triage_positions ORDER BY updated_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("store: query triage positions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var positions []domain.TriagePosition
	for rows.Next() {
		var pos domain.TriagePosition
		var updatedAt int64
		if err := rows.Scan(&pos.ResourceID, &pos.Bucket, &updatedAt); err != nil {
			return nil, fmt.Errorf("store: scan triage position: %w", err)
		}
		pos.UpdatedAt = time.Unix(updatedAt, 0)
		positions = append(positions, pos)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: rows iteration: %w", err)
	}

	return positions, nil
}
