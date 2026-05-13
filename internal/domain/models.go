package domain

import "time"

// Bucket represents a triage bucket for resources.
type Bucket string

const (
	BucketInbox    Bucket = "inbox"
	BucketActive   Bucket = "active"
	BucketLater    Bucket = "later"
	BucketArchived Bucket = "archived"
)

// AllBuckets returns all valid bucket values.
func AllBuckets() []Bucket {
	return []Bucket{BucketInbox, BucketActive, BucketLater, BucketArchived}
}

// Valid reports whether b is a recognized bucket.
func (b Bucket) Valid() bool {
	switch b {
	case BucketInbox, BucketActive, BucketLater, BucketArchived:
		return true
	}
	return false
}

// Resource represents a knowledge resource tracked by Marrow.
type Resource struct {
	ID        string    `json:"id"`
	URL       string    `json:"url"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Bucket    Bucket    `json:"bucket"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TriagePosition tracks the local bucket assignment for a resource.
type TriagePosition struct {
	ResourceID string    `json:"resource_id"`
	Bucket     Bucket    `json:"bucket"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// SearchCache holds a cached search result with TTL.
type SearchCache struct {
	Query     string    `json:"query"`
	Results   string    `json:"results"` // JSON-encoded []Resource
	CachedAt  time.Time `json:"cached_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// IsExpired reports whether the cache entry has passed its TTL.
func (c *SearchCache) IsExpired() bool {
	return time.Now().After(c.ExpiresAt)
}
