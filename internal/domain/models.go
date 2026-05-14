package domain

import (
	"regexp"
	"strings"
	"time"
)

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

// NodeType represents the kind of graph node.
type NodeType string

const (
	NodeTypeDomain   NodeType = "domain"
	NodeTypeSubarea  NodeType = "subarea"
	NodeTypeProject  NodeType = "project"
	NodeTypeSession  NodeType = "session"
	NodeTypeLearning NodeType = "learning"
	NodeTypePerson   NodeType = "person"
)

// Valid reports whether nt is a recognized node type.
func (nt NodeType) Valid() bool {
	switch nt {
	case NodeTypeDomain, NodeTypeSubarea, NodeTypeProject, NodeTypeSession, NodeTypeLearning, NodeTypePerson:
		return true
	}
	return false
}

// AllNodeTypes returns all valid node type values.
func AllNodeTypes() []NodeType {
	return []NodeType{NodeTypeDomain, NodeTypeSubarea, NodeTypeProject, NodeTypeSession, NodeTypeLearning, NodeTypePerson}
}

// EdgeLabel defines valid graph edge types.
type EdgeLabel string

const (
	EdgeContains      EdgeLabel = "contains"       // Domain → Subarea
	EdgeAppliesTo     EdgeLabel = "applies_to"     // Project → Subarea
	EdgeWorkedOn      EdgeLabel = "worked_on"      // Project → Session
	EdgeLearnedFrom   EdgeLabel = "learned_from"   // Learning → Session
	EdgeReferences    EdgeLabel = "references"     // Resource → Subarea
)

// GraphNode is the SQLite-cached node topology record.
type GraphNode struct {
	EngramID string    `json:"engram_id"`
	NodeType NodeType  `json:"node_type"`
	Title    string    `json:"title"`
	Active   bool      `json:"active"`
	CachedAt time.Time `json:"cached_at"`
}

// GraphEdge represents a directed labeled edge.
type GraphEdge struct {
	FromID string     `json:"from_id"`
	ToID   string     `json:"to_id"`
	Label  EdgeLabel  `json:"label"`
}

// Domain represents a user-defined knowledge domain.
type Domain struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Active      bool   `json:"active"`
}

// Subarea represents a subdivision within a Domain.
type Subarea struct {
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	DomainID string `json:"domain_id"` // Engram ID of parent Domain
	Active   bool   `json:"active"`
}

// Project represents a learning/project tracked within Subareas.
type Project struct {
	Name        string   `json:"name"`
	Slug        string   `json:"slug"`
	Description string   `json:"description"`
	Active      bool     `json:"active"`
	SubareaIDs  []string `json:"subarea_ids"` // Engram IDs of linked Subareas
}

// DomainWithSubareas is a read-only projection for tree rendering.
type DomainWithSubareas struct {
	Domain   GraphNode   `json:"domain"`
	Subareas []GraphNode `json:"subareas"`
}

// Session represents a learning or work session tracked within a Project.
type Session struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

// Learning represents a knowledge artifact captured during a Session.
type Learning struct {
	ID          string    `json:"id"`
	SessionID   string    `json:"session_id"`
	Content     string    `json:"content"`
	SubareaIDs  []string  `json:"subarea_ids"`
	ProjectIDs  []string  `json:"project_ids"`
	CreatedAt   time.Time `json:"created_at"`
}

// Person represents a user tracked in the knowledge graph.
type Person struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

// Slugify generates a deterministic slug from a name.
// Lowercase, spaces → hyphens, non-alphanumeric stripped.
// "Backend Development" → "backend-development"
func Slugify(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	// Replace spaces with hyphens
	s = strings.ReplaceAll(s, " ", "-")
	// Remove non-alphanumeric characters except hyphens
	re := regexp.MustCompile(`[^a-z0-9-]`)
	s = re.ReplaceAllString(s, "")
	// Collapse multiple hyphens
	re2 := regexp.MustCompile(`-+`)
	s = re2.ReplaceAllString(s, "-")
	// Trim leading/trailing hyphens
	s = strings.Trim(s, "-")
	return s
}
