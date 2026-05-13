package domain

import "context"

// EngramClient is the interface for interacting with the Engram knowledge graph API.
type EngramClient interface {
	CreateResource(ctx context.Context, r Resource) (string, error)
	GetResource(ctx context.Context, id string) (Resource, error)
	SearchResources(ctx context.Context, query string) ([]Resource, error)
	GetRoadmap(ctx context.Context) (map[Bucket][]Resource, error)
	UpdateResource(ctx context.Context, id string, updates map[string]any) error
	IsReachable(ctx context.Context) bool
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

// Config holds Marrow configuration values.
type Config struct {
	Profile    string `yaml:"profile"`
	EngramPath string `yaml:"engram_path"`
	ModelPrefs struct {
		Triage  string `yaml:"triage"`
		Summary string `yaml:"summary"`
	} `yaml:"model_prefs"`
}
