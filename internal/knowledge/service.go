package knowledge

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Nick-2455/silo/internal/knowledge/notemodel"
)

// EngramReader reads knowledge items from Engram.
type EngramReader interface {
	ReadKnowledge(ctx context.Context, q EngramQuery) ([]KnowledgeItem, error)
}

// Vault writes and searches Markdown notes inside an Obsidian vault.
type Vault interface {
	CreateOrUpdateNote(ctx context.Context, vaultPath string, note Note) (NoteWriteResult, error)
	SearchVault(ctx context.Context, vaultPath, query string, limit int) ([]NoteSearchResult, error)
}

// EngramQuery filters Engram knowledge reads.
type EngramQuery struct {
	Query   string `json:"query"`
	Project string `json:"project,omitempty"`
	Type    string `json:"type,omitempty"`
	Limit   int    `json:"limit,omitempty"`
}

// SyncRequest drives an Engram → Obsidian sync run.
type SyncRequest struct {
	Engram    EngramQuery
	VaultPath string
}

// SyncReport summarizes a sync operation.
type SyncReport struct {
	Total   int      `json:"total"`
	Written int      `json:"written"`
	Updated int      `json:"updated"`
	Errors  []string `json:"errors,omitempty"`
}

// ContextRequest collects Engram knowledge and vault hits for an agent.
type ContextRequest struct {
	Query     string
	Project   string
	VaultPath string
	Limit     int
}

// Service coordinates Engram reads, vault writes/search, and context assembly.
type Service struct {
	Engram EngramReader
	Vault  Vault
}

// NewService builds a Service with required dependencies.
func NewService(reader EngramReader, vault Vault) *Service {
	return &Service{Engram: reader, Vault: vault}
}

// ReadKnowledge proxies an Engram read.
func (s *Service) ReadKnowledge(ctx context.Context, q EngramQuery) ([]KnowledgeItem, error) {
	if s.Engram == nil {
		return nil, errors.New("knowledge: engram reader is not configured")
	}
	return s.Engram.ReadKnowledge(ctx, q)
}

// SyncToObsidian reads Engram knowledge and writes Markdown notes into the vault.
func (s *Service) SyncToObsidian(ctx context.Context, req SyncRequest) (*SyncReport, error) {
	if s.Engram == nil {
		return nil, errors.New("knowledge: engram reader is not configured")
	}
	if s.Vault == nil {
		return nil, errors.New("knowledge: vault is not configured")
	}
	if strings.TrimSpace(req.VaultPath) == "" {
		return nil, errors.New("knowledge: vault path is required")
	}

	items, err := s.Engram.ReadKnowledge(ctx, req.Engram)
	if err != nil {
		return nil, fmt.Errorf("engram read: %w", err)
	}

	report := &SyncReport{Total: len(items)}
	for _, item := range items {
		note := noteFromKnowledge(item)
		result, err := s.Vault.CreateOrUpdateNote(ctx, req.VaultPath, note)
		if err != nil {
			report.Errors = append(report.Errors, fmt.Sprintf("%s: %v", item.Title, err))
			continue
		}
		if result.Created {
			report.Written++
		} else {
			report.Updated++
		}
	}
	return report, nil
}

// SearchVault searches the Markdown vault.
func (s *Service) SearchVault(ctx context.Context, vaultPath, query string, limit int) ([]NoteSearchResult, error) {
	if s.Vault == nil {
		return nil, errors.New("knowledge: vault is not configured")
	}
	return s.Vault.SearchVault(ctx, vaultPath, query, limit)
}

// CreateOrUpdateNote writes a single note safely.
// When note.Type is non-empty, community note-model defaults are merged into
// the frontmatter before the note is written.
func (s *Service) CreateOrUpdateNote(ctx context.Context, vaultPath string, note Note) (NoteWriteResult, error) {
	if s.Vault == nil {
		return NoteWriteResult{}, errors.New("knowledge: vault is not configured")
	}
	if note.Type != "" {
		note.Frontmatter = notemodel.ApplyDefaults(
			notemodel.Type(note.Type),
			note.Kind,
			note.Frontmatter,
		)
	}
	return s.Vault.CreateOrUpdateNote(ctx, vaultPath, note)
}

// GetKnowledgeContext combines Engram knowledge with vault matches.
func (s *Service) GetKnowledgeContext(ctx context.Context, req ContextRequest) (*KnowledgeContext, error) {
	if strings.TrimSpace(req.Query) == "" {
		return nil, errors.New("knowledge: query is required")
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}

	context := &KnowledgeContext{Query: req.Query}

	if s.Engram != nil {
		items, err := s.Engram.ReadKnowledge(ctx, EngramQuery{
			Query:   req.Query,
			Project: req.Project,
			Limit:   limit,
		})
		if err == nil {
			context.Engram = items
		} else {
			return nil, fmt.Errorf("engram read: %w", err)
		}
	}

	if s.Vault != nil && strings.TrimSpace(req.VaultPath) != "" {
		matches, err := s.Vault.SearchVault(ctx, req.VaultPath, req.Query, limit)
		if err == nil {
			context.Vault = matches
		}
	}

	return context, nil
}

// noteFromKnowledge converts an Engram knowledge item into a vault note.
// When the item carries a recognized community note type, notemodel defaults
// are applied to enrich the frontmatter.
func noteFromKnowledge(item KnowledgeItem) Note {
	frontmatter := map[string]any{
		"source": "engram",
	}
	if item.ID != "" {
		frontmatter["engram_id"] = item.ID
	}
	if item.Type != "" {
		frontmatter["type"] = item.Type
	}
	if item.Project != "" {
		frontmatter["project"] = item.Project
	}

	noteType := notemodel.Type(item.Type)
	if notemodel.ValidType(noteType) {
		frontmatter = notemodel.ApplyDefaults(noteType, "", frontmatter)
	}

	content := item.Content
	if content == "" {
		content = item.Preview
	}
	return Note{
		Title:       item.Title,
		Content:     content,
		Frontmatter: frontmatter,
	}
}
