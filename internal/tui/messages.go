package tui

import (
	"github.com/Nick-2455/silo/internal/domain"
	"github.com/Nick-2455/silo/internal/obsidian"
)

// tickMsg triggers periodic health checks.
type tickMsg struct{}

// windowSizeMsg handles terminal resize (wraps tea.WindowSizeMsg to avoid import in messages).
type windowSizeMsg struct {
	Width  int
	Height int
}

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

// SyncDoneMsg is sent when an Obsidian sync completes.
type SyncDoneMsg struct {
	Report *obsidian.SyncReport
	Err    string
}

// DomainTreeLoadedMsg carries the result of loading the domain tree.
type DomainTreeLoadedMsg struct {
	Tree []domain.DomainWithSubareas
	Err  string
}

// ProjectsLoadedMsg carries the result of loading projects.
type ProjectsLoadedMsg struct {
	Projects []domain.Project
	Err      string
}

// SessionsLoadedMsg carries the result of loading sessions.
type SessionsLoadedMsg struct {
	Sessions []domain.Session
	Err      string
}

// LearningsLoadedMsg carries the result of loading learnings.
type LearningsLoadedMsg struct {
	Learnings []domain.Learning
	Err       string
}
