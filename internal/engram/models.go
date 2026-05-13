package engram

import "time"

// Observation represents an Engram observation (the underlying type for Resources).
type Observation struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Content   map[string]interface{} `json:"content"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// Relation represents a directed edge between two observations.
type Relation struct {
	ID         string    `json:"id"`
	SourceID   string    `json:"source_id"`
	TargetID   string    `json:"target_id"`
	Type       string    `json:"type"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// SearchRequest is the payload for FTS5 search queries.
type SearchRequest struct {
	Query string `json:"query"`
	Limit int    `json:"limit,omitempty"`
}

// SearchResponse is the response from an Engram search.
type SearchResponse struct {
	Results []Observation `json:"results"`
	Total   int           `json:"total"`
}
