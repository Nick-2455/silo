package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Nick-2455/marrow/internal/domain"
)

// Verify *Store implements domain.GraphStore.
var _ domain.GraphStore = (*Store)(nil)

// UpsertNode inserts or replaces a graph node in SQLite.
func (s *Store) UpsertNode(_ context.Context, node domain.GraphNode) error {
	active := 0
	if node.Active {
		active = 1
	}
	_, err := s.db.Exec(
		`INSERT INTO graph_nodes (engram_id, node_type, title, active, cached_at)
		 VALUES (?, ?, ?, ?, datetime('now'))
		 ON CONFLICT(engram_id) DO UPDATE SET
			node_type = excluded.node_type,
			title = excluded.title,
			active = excluded.active,
			cached_at = datetime('now')`,
		node.EngramID, node.NodeType, node.Title, active,
	)
	if err != nil {
		return fmt.Errorf("store: upsert node: %w", err)
	}
	return nil
}

// DeleteNode soft-deletes a node by setting active=0.
func (s *Store) DeleteNode(_ context.Context, engramID string) error {
	result, err := s.db.Exec(
		`UPDATE graph_nodes SET active = 0 WHERE engram_id = ? AND active = 1`,
		engramID,
	)
	if err != nil {
		return fmt.Errorf("store: delete node: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("store: delete node rows: %w", err)
	}
	if rows == 0 {
		return domain.ErrNodeNotFound
	}
	return nil
}

// GetNode retrieves a single graph node by Engram ID.
func (s *Store) GetNode(_ context.Context, engramID string) (domain.GraphNode, error) {
	var node domain.GraphNode
	var active int64
	var cachedAt string

	err := s.db.QueryRow(
		`SELECT engram_id, node_type, title, active, cached_at FROM graph_nodes WHERE engram_id = ?`,
		engramID,
	).Scan(&node.EngramID, &node.NodeType, &node.Title, &active, &cachedAt)

	if err == sql.ErrNoRows {
		return node, domain.ErrNodeNotFound
	}
	if err != nil {
		return node, fmt.Errorf("store: get node: %w", err)
	}

	node.Active = active == 1
	node.CachedAt = parseTime(cachedAt)
	return node, nil
}

// ListNodesByType returns all nodes of a given type.
func (s *Store) ListNodesByType(_ context.Context, nodeType domain.NodeType) ([]domain.GraphNode, error) {
	rows, err := s.db.Query(
		`SELECT engram_id, node_type, title, active, cached_at FROM graph_nodes WHERE node_type = ? ORDER BY title`,
		nodeType,
	)
	if err != nil {
		return nil, fmt.Errorf("store: list nodes by type: %w", err)
	}
	defer func() { _ = rows.Close() }()

	return scanGraphNodes(rows)
}

// AddEdge inserts a directed labeled edge. Returns ErrDuplicateNode if the edge already exists.
func (s *Store) AddEdge(_ context.Context, fromID, toID string, label domain.EdgeLabel) error {
	_, err := s.db.Exec(
		`INSERT INTO graph_edges (from_id, to_id, label) VALUES (?, ?, ?)`,
		fromID, toID, label,
	)
	if err != nil {
		// SQLite UNIQUE constraint violation
		if isUniqueConstraintViolation(err) {
			return domain.ErrDuplicateNode
		}
		return fmt.Errorf("store: add edge: %w", err)
	}
	return nil
}

// RemoveEdge deletes a specific edge.
func (s *Store) RemoveEdge(_ context.Context, fromID, toID string, label domain.EdgeLabel) error {
	_, err := s.db.Exec(
		`DELETE FROM graph_edges WHERE from_id = ? AND to_id = ? AND label = ?`,
		fromID, toID, label,
	)
	if err != nil {
		return fmt.Errorf("store: remove edge: %w", err)
	}
	return nil
}

// GetEdges returns edges for a node. Direction: "from", "to", or "both".
func (s *Store) GetEdges(_ context.Context, nodeID string, direction string) ([]domain.GraphEdge, error) {
	var rows *sql.Rows
	var err error

	switch direction {
	case "from":
		rows, err = s.db.Query(
			`SELECT from_id, to_id, label FROM graph_edges WHERE from_id = ?`,
			nodeID,
		)
	case "to":
		rows, err = s.db.Query(
			`SELECT from_id, to_id, label FROM graph_edges WHERE to_id = ?`,
			nodeID,
		)
	case "both":
		rows, err = s.db.Query(
			`SELECT from_id, to_id, label FROM graph_edges WHERE from_id = ? OR to_id = ?`,
			nodeID, nodeID,
		)
	default:
		return nil, fmt.Errorf("store: invalid direction %q, must be from/to/both", direction)
	}
	if err != nil {
		return nil, fmt.Errorf("store: get edges: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var edges []domain.GraphEdge
	for rows.Next() {
		var e domain.GraphEdge
		if err := rows.Scan(&e.FromID, &e.ToID, &e.Label); err != nil {
			return nil, fmt.Errorf("store: scan edge: %w", err)
		}
		edges = append(edges, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: edges iteration: %w", err)
	}
	return edges, nil
}

// GetNeighbors returns nodes connected to nodeID via edges with the given label.
func (s *Store) GetNeighbors(_ context.Context, nodeID string, label domain.EdgeLabel) ([]domain.GraphNode, error) {
	rows, err := s.db.Query(
		`SELECT DISTINCT gn.engram_id, gn.node_type, gn.title, gn.active, gn.cached_at
		 FROM graph_nodes gn
		 JOIN graph_edges ge ON gn.engram_id = ge.to_id OR gn.engram_id = ge.from_id
		 WHERE (ge.from_id = ? OR ge.to_id = ?) AND ge.label = ? AND gn.engram_id != ?`,
		nodeID, nodeID, label, nodeID,
	)
	if err != nil {
		return nil, fmt.Errorf("store: get neighbors: %w", err)
	}
	defer func() { _ = rows.Close() }()

	return scanGraphNodes(rows)
}

// GetDomainTree returns all domains with their nested subareas.
func (s *Store) GetDomainTree(ctx context.Context) ([]domain.DomainWithSubareas, error) {
	domains, err := s.ListNodesByType(ctx, domain.NodeTypeDomain)
	if err != nil {
		return nil, fmt.Errorf("store: get domain tree: %w", err)
	}

	var result []domain.DomainWithSubareas
	for _, d := range domains {
		dws := domain.DomainWithSubareas{Domain: d}

		// Find subareas connected via "contains" edges from this domain
		rows, err := s.db.Query(
			`SELECT gn.engram_id, gn.node_type, gn.title, gn.active, gn.cached_at
			 FROM graph_nodes gn
			 JOIN graph_edges ge ON gn.engram_id = ge.to_id
			 WHERE ge.from_id = ? AND ge.label = ?`,
			d.EngramID, domain.EdgeContains,
		)
		if err != nil {
			return nil, fmt.Errorf("store: query subareas for domain %s: %w", d.EngramID, err)
		}

		subareas, scanErr := scanGraphNodes(rows)
		_ = rows.Close()
		if scanErr != nil {
			return nil, fmt.Errorf("store: scan subareas for domain %s: %w", d.EngramID, scanErr)
		}
		dws.Subareas = subareas
		result = append(result, dws)
	}

	return result, nil
}

// ListActiveProjects returns all projects with active=1.
func (s *Store) ListActiveProjects(_ context.Context) ([]domain.Project, error) {
	rows, err := s.db.Query(
		`SELECT engram_id, title FROM graph_nodes
		 WHERE node_type = ? AND active = 1 ORDER BY title`,
		domain.NodeTypeProject,
	)
	if err != nil {
		return nil, fmt.Errorf("store: list active projects: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var projects []domain.Project
	for rows.Next() {
		var engramID, title string
		if err := rows.Scan(&engramID, &title); err != nil {
			return nil, fmt.Errorf("store: scan project: %w", err)
		}

		// Fetch subarea links via "applies_to" edges
		subareaRows, err := s.db.Query(
			`SELECT to_id FROM graph_edges
			 WHERE from_id = ? AND label = ?`,
			engramID, domain.EdgeAppliesTo,
		)
		if err != nil {
			return nil, fmt.Errorf("store: query project subareas: %w", err)
		}

		var subareaIDs []string
		for subareaRows.Next() {
			var sid string
			if err := subareaRows.Scan(&sid); err != nil {
				_ = subareaRows.Close()
				return nil, fmt.Errorf("store: scan project subarea: %w", err)
			}
			subareaIDs = append(subareaIDs, sid)
		}
		_ = subareaRows.Close()

		projects = append(projects, domain.Project{
			Name:       title,
			Slug:       domain.Slugify(title),
			Active:     true,
			SubareaIDs: subareaIDs,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: projects iteration: %w", err)
	}
	return projects, nil
}

// UpsertPerson inserts or replaces a person node. Used for auto-creating the default user.
func (s *Store) UpsertPerson(_ context.Context, node domain.GraphNode) error {
	return s.UpsertNode(context.Background(), node)
}

// ListSessions returns sessions for a given project.
func (s *Store) ListSessions(_ context.Context, projectID string) ([]domain.Session, error) {
	// Edge direction: Session --worked_on--> Project
	// Find sessions where the edge points TO this project.
	rows, err := s.db.Query(
		`SELECT gn.engram_id, gn.title, gn.cached_at
		 FROM graph_nodes gn
		 JOIN graph_edges ge ON gn.engram_id = ge.from_id
		 WHERE ge.to_id = ? AND ge.label = ? AND gn.node_type = ? AND gn.active = 1
		 ORDER BY gn.cached_at DESC`,
		projectID, domain.EdgeWorkedOn, domain.NodeTypeSession,
	)
	if err != nil {
		return nil, fmt.Errorf("store: list sessions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var sessions []domain.Session
	for rows.Next() {
		var id, title, cachedAt string
		if err := rows.Scan(&id, &title, &cachedAt); err != nil {
			return nil, fmt.Errorf("store: scan session: %w", err)
		}
		sessions = append(sessions, domain.Session{
			ID:          id,
			ProjectID:   projectID,
			Description: title,
			CreatedAt:   parseTime(cachedAt),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: sessions iteration: %w", err)
	}
	return sessions, nil
}

// GetSession retrieves a session by ID.
func (s *Store) GetSession(_ context.Context, id string) (domain.Session, error) {
	var session domain.Session
	var title, cachedAt string

	err := s.db.QueryRow(
		`SELECT engram_id, title, cached_at FROM graph_nodes WHERE engram_id = ? AND node_type = ?`,
		id, domain.NodeTypeSession,
	).Scan(&session.ID, &title, &cachedAt)

	if err == sql.ErrNoRows {
		return session, domain.ErrSessionNotFound
	}
	if err != nil {
		return session, fmt.Errorf("store: get session: %w", err)
	}

	session.Description = title
	session.CreatedAt = parseTime(cachedAt)

	// Find project via worked_on edge (session is the "to" side)
	var projectID sql.NullString
	_ = s.db.QueryRow(
		`SELECT from_id FROM graph_edges WHERE to_id = ? AND label = ?`,
		id, domain.EdgeWorkedOn,
	).Scan(&projectID)
	if projectID.Valid {
		session.ProjectID = projectID.String
	}

	return session, nil
}

// ListLearnings returns learnings, optionally filtered by subarea.
func (s *Store) ListLearnings(_ context.Context, subareaID string) ([]domain.Learning, error) {
	var rows *sql.Rows
	var err error

	if subareaID != "" {
		// Edge direction: Learning --applies_to--> Subarea
		// Find learnings where the edge points TO this subarea.
		rows, err = s.db.Query(
			`SELECT DISTINCT gn.engram_id, gn.title, gn.cached_at
			 FROM graph_nodes gn
			 JOIN graph_edges ge ON gn.engram_id = ge.from_id
			 WHERE ge.to_id = ? AND ge.label = ? AND gn.node_type = ? AND gn.active = 1
			 ORDER BY gn.cached_at DESC`,
			subareaID, domain.EdgeAppliesTo, domain.NodeTypeLearning,
		)
	} else {
		rows, err = s.db.Query(
			`SELECT engram_id, title, cached_at FROM graph_nodes
			 WHERE node_type = ? AND active = 1 ORDER BY cached_at DESC`,
			domain.NodeTypeLearning,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("store: list learnings: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var learnings []domain.Learning
	for rows.Next() {
		var id, content, cachedAt string
		if err := rows.Scan(&id, &content, &cachedAt); err != nil {
			return nil, fmt.Errorf("store: scan learning: %w", err)
		}
		l := domain.Learning{
			ID:        id,
			Content:   content,
			CreatedAt: parseTime(cachedAt),
		}

		// Fetch linked subareas via references edges (learning → subarea)
		subRows, serr := s.db.Query(
			`SELECT to_id FROM graph_edges WHERE from_id = ? AND label = ?`,
			id, domain.EdgeReferences,
		)
		if serr == nil {
			for subRows.Next() {
				var sid string
				if err := subRows.Scan(&sid); err == nil {
					l.SubareaIDs = append(l.SubareaIDs, sid)
				}
			}
			_ = subRows.Close()
		}

		// Fetch linked sessions via learned_from edges (learning → session)
		sessRows, serr := s.db.Query(
			`SELECT to_id FROM graph_edges WHERE from_id = ? AND label = ?`,
			id, domain.EdgeLearnedFrom,
		)
		if serr == nil {
			for sessRows.Next() {
				var sid string
				if err := sessRows.Scan(&sid); err == nil {
					l.SessionID = sid
					break // one session per learning
				}
			}
			_ = sessRows.Close()
		}

		learnings = append(learnings, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: learnings iteration: %w", err)
	}
	return learnings, nil
}

// GetLearning retrieves a learning by ID.
func (s *Store) GetLearning(_ context.Context, id string) (domain.Learning, error) {
	var learning domain.Learning
	var content, cachedAt string

	err := s.db.QueryRow(
		`SELECT engram_id, title, cached_at FROM graph_nodes WHERE engram_id = ? AND node_type = ?`,
		id, domain.NodeTypeLearning,
	).Scan(&learning.ID, &content, &cachedAt)

	if err == sql.ErrNoRows {
		return learning, domain.ErrLearningNotFound
	}
	if err != nil {
		return learning, fmt.Errorf("store: get learning: %w", err)
	}

	learning.Content = content
	learning.CreatedAt = parseTime(cachedAt)

	// Fetch linked subareas
	subRows, err := s.db.Query(
		`SELECT to_id FROM graph_edges WHERE from_id = ? AND label = ?`,
		id, domain.EdgeReferences,
	)
	if err == nil {
		for subRows.Next() {
			var sid string
			if err := subRows.Scan(&sid); err == nil {
				learning.SubareaIDs = append(learning.SubareaIDs, sid)
			}
		}
		_ = subRows.Close()
	}

	// Fetch linked session
	var sessionID sql.NullString
	_ = s.db.QueryRow(
		`SELECT to_id FROM graph_edges WHERE from_id = ? AND label = ?`,
		id, domain.EdgeLearnedFrom,
	).Scan(&sessionID)
	if sessionID.Valid {
		learning.SessionID = sessionID.String
	}

	return learning, nil
}

// scanGraphNodes scans rows into a slice of GraphNode.
func scanGraphNodes(rows *sql.Rows) ([]domain.GraphNode, error) {
	if rows == nil {
		return []domain.GraphNode{}, nil
	}

	var nodes []domain.GraphNode
	for rows.Next() {
		var node domain.GraphNode
		var active int64
		var cachedAt string
		if err := rows.Scan(&node.EngramID, &node.NodeType, &node.Title, &active, &cachedAt); err != nil {
			return nil, fmt.Errorf("store: scan graph node: %w", err)
		}
		node.Active = active == 1
		node.CachedAt = parseTime(cachedAt)
		nodes = append(nodes, node)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: nodes iteration: %w", err)
	}
	return nodes, nil
}

// parseTime parses a SQLite datetime string into time.Time.
// SQLite returns "2006-01-02 15:04:05" or "2006-01-02T15:04:05Z" format.
func parseTime(s string) time.Time {
	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		time.RFC3339,
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

// isUniqueConstraintViolation checks if the error is a SQLite UNIQUE constraint violation.
func isUniqueConstraintViolation(err error) bool {
	// modernc.org/sqlite returns errors containing "UNIQUE constraint failed"
	return err != nil && (containsStr(err.Error(), "UNIQUE constraint failed") ||
		containsStr(err.Error(), "UNIQUE"))
}

// containsStr is a simple string contains check without importing strings.
func containsStr(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
