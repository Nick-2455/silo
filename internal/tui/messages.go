package tui

import "github.com/Nick-2455/marrow/internal/domain"

// tickMsg triggers periodic health checks.
type tickMsg struct{}

// HealthResultMsg carries the result of a health check.
type HealthResultMsg struct {
	OK bool
}

// ResourceLoadedMsg carries loaded resources from the store.
type ResourceLoadedMsg struct {
	Resources []domain.Resource
	Err       string
}

// AddSubmitMsg is sent when the add form submission completes.
type AddSubmitMsg struct {
	ID  string
	Err string
}

// TriageMoveMsg is sent when a triage move operation completes.
type TriageMoveMsg struct {
	Bucket domain.Bucket
	Err    string
}
