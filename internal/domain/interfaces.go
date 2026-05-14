package domain

import "context"

// EngramClient is the interface for interacting with the Engram knowledge graph API.
type EngramClient interface {
	CreateResource(ctx context.Context, r Resource) (string, error)
	GetResource(ctx context.Context, id string) (Resource, error)
	SearchResources(ctx context.Context, query string) ([]Resource, error)
	UpdateResource(ctx context.Context, id string, updates map[string]any) error
	IsReachable(ctx context.Context) bool
	SaveNode(ctx context.Context, nodeType, title string, content map[string]any, topicKey string) (string, error)
	UpdateNode(ctx context.Context, engramID string, content map[string]any) error
}

// ResourceStore is the interface for local SQLite persistence.
type ResourceStore interface {
	GetTriagePosition(ctx context.Context, resourceID string) (TriagePosition, error)
	SetTriagePosition(ctx context.Context, pos TriagePosition) error
	GetAllTriagePositions(ctx context.Context) ([]TriagePosition, error)
	CacheSearch(ctx context.Context, query string, results []Resource) error
	GetCachedSearch(ctx context.Context, query string) ([]Resource, bool, error)
	InvalidateSearchCache(ctx context.Context) error
	Close() error
}

// ConfigLoader is the interface for loading and saving Marrow configuration.
type ConfigLoader interface {
	Load() (Config, error)
	Save(cfg Config) error
	Path() string
}

// GraphStore manages graph nodes and edges in SQLite.
type GraphStore interface {
	// Node operations
	UpsertNode(ctx context.Context, node GraphNode) error
	DeleteNode(ctx context.Context, engramID string) error // soft delete
	GetNode(ctx context.Context, engramID string) (GraphNode, error)
	ListNodesByType(ctx context.Context, nodeType NodeType) ([]GraphNode, error)

	// Edge operations
	AddEdge(ctx context.Context, fromID, toID string, label EdgeLabel) error
	RemoveEdge(ctx context.Context, fromID, toID string, label EdgeLabel) error
	GetEdges(ctx context.Context, nodeID string, direction string) ([]GraphEdge, error) // direction: "from", "to", "both"
	GetNeighbors(ctx context.Context, nodeID string, label EdgeLabel) ([]GraphNode, error)

	// Domain-specific queries
	GetDomainTree(ctx context.Context) ([]DomainWithSubareas, error)
	ListActiveProjects(ctx context.Context) ([]Project, error)

	// Session operations
	ListSessions(ctx context.Context, projectID string) ([]Session, error)
	GetSession(ctx context.Context, id string) (Session, error)

	// Learning operations
	ListLearnings(ctx context.Context, subareaID string) ([]Learning, error)
	GetLearning(ctx context.Context, id string) (Learning, error)

	// Person operations
	UpsertPerson(ctx context.Context, node GraphNode) error

	// Close shares the Store's db; no-op
	Close() error
}

// Config holds Marrow configuration values.
type Config struct {
	Profile    string `yaml:"profile"`
	EngramPath string `yaml:"engram_path"`
	ModelPrefs struct {
		Triage  string `yaml:"triage"`
		Summary string `yaml:"summary"`
	} `yaml:"model_prefs"`
}
