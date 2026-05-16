package knowledge

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Nick-2455/silo/internal/knowledge/notemodel"
)

type fakeReader struct {
	items []KnowledgeItem
	err   error
	last  EngramQuery
}

func (f *fakeReader) ReadKnowledge(_ context.Context, q EngramQuery) ([]KnowledgeItem, error) {
	f.last = q
	if f.err != nil {
		return nil, f.err
	}
	return f.items, nil
}

func TestService_ReadKnowledgeProxiesEngram(t *testing.T) {
	reader := &fakeReader{items: []KnowledgeItem{{Title: "A"}, {Title: "B"}}}
	svc := NewService(reader, VaultStore{})

	got, err := svc.ReadKnowledge(context.Background(), EngramQuery{Query: "hello"})
	if err != nil {
		t.Fatalf("read knowledge: %v", err)
	}
	if len(got) != 2 || reader.last.Query != "hello" {
		t.Fatalf("unexpected read result: %+v / %+v", got, reader.last)
	}
}

func TestService_ReadKnowledgeFailsWithoutEngram(t *testing.T) {
	svc := NewService(nil, VaultStore{})
	if _, err := svc.ReadKnowledge(context.Background(), EngramQuery{Query: "q"}); err == nil {
		t.Fatalf("expected error when engram reader missing")
	}
}

func TestService_SyncToObsidianWritesNotes(t *testing.T) {
	vault := t.TempDir()
	reader := &fakeReader{items: []KnowledgeItem{
		{ID: "1", Title: "Engram Memory", Content: "Engram persists memory.", Type: "learning"},
		{ID: "2", Title: "Silo Bridge", Content: "Silo bridges Markdown."},
	}}
	svc := NewService(reader, VaultStore{})

	report, err := svc.SyncToObsidian(context.Background(), SyncRequest{
		Engram:    EngramQuery{Query: "memory"},
		VaultPath: vault,
	})
	if err != nil {
		t.Fatalf("sync: %v", err)
	}

	if report.Total != 2 || report.Written != 2 || report.Updated != 0 {
		t.Fatalf("unexpected report: %+v", report)
	}

	expected := filepath.Join(vault, "Silo", "Knowledge", "engram-memory.md")
	if _, err := os.Stat(expected); err != nil {
		t.Fatalf("expected note at %s: %v", expected, err)
	}

	again, err := svc.SyncToObsidian(context.Background(), SyncRequest{
		Engram:    EngramQuery{Query: "memory"},
		VaultPath: vault,
	})
	if err != nil {
		t.Fatalf("second sync: %v", err)
	}
	if again.Written != 0 || again.Updated != 2 {
		t.Fatalf("second sync should update existing notes: %+v", again)
	}
}

func TestService_SyncToObsidianRequiresVaultPath(t *testing.T) {
	svc := NewService(&fakeReader{}, VaultStore{})
	if _, err := svc.SyncToObsidian(context.Background(), SyncRequest{}); err == nil {
		t.Fatalf("expected error on empty vault path")
	}
}

func TestService_GetKnowledgeContextCombinesSources(t *testing.T) {
	vault := t.TempDir()
	if _, err := (VaultStore{}).CreateOrUpdateNote(context.Background(), vault, Note{
		Title:   "Engram Memory",
		Content: "Engram persists durable memory.",
	}); err != nil {
		t.Fatalf("seed note: %v", err)
	}

	reader := &fakeReader{items: []KnowledgeItem{{Title: "Cached", Content: "From Engram"}}}
	svc := NewService(reader, VaultStore{})

	ctxResult, err := svc.GetKnowledgeContext(context.Background(), ContextRequest{
		Query:     "durable memory",
		VaultPath: vault,
	})
	if err != nil {
		t.Fatalf("knowledge context: %v", err)
	}

	if len(ctxResult.Engram) != 1 {
		t.Fatalf("expected 1 engram item, got %d", len(ctxResult.Engram))
	}
	if len(ctxResult.Vault) != 1 {
		t.Fatalf("expected 1 vault hit, got %d", len(ctxResult.Vault))
	}
	if !strings.Contains(ctxResult.Vault[0].Snippet, "durable memory") {
		t.Fatalf("unexpected vault snippet: %q", ctxResult.Vault[0].Snippet)
	}
}

func TestService_GetKnowledgeContextRequiresQuery(t *testing.T) {
	svc := NewService(&fakeReader{}, VaultStore{})
	if _, err := svc.GetKnowledgeContext(context.Background(), ContextRequest{}); err == nil {
		t.Fatalf("expected error on empty query")
	}
}

func TestService_GetKnowledgeContextSurfacesEngramErrors(t *testing.T) {
	reader := &fakeReader{err: errors.New("engram offline")}
	svc := NewService(reader, VaultStore{})

	_, err := svc.GetKnowledgeContext(context.Background(), ContextRequest{Query: "x"})
	if err == nil || !strings.Contains(err.Error(), "engram offline") {
		t.Fatalf("expected engram error to surface, got: %v", err)
	}
}

// --- notemodel integration tests ---

type fakeVault struct {
	lastNote Note
}

func (f *fakeVault) CreateOrUpdateNote(_ context.Context, _ string, note Note) (NoteWriteResult, error) {
	f.lastNote = note
	return NoteWriteResult{Path: "fake.md", Created: true}, nil
}

func (f *fakeVault) SearchVault(_ context.Context, _, _ string, _ int) ([]NoteSearchResult, error) {
	return nil, nil
}

func TestService_CreateOrUpdateNote_AppliesDefaultsWhenTypeSet(t *testing.T) {
	vault := &fakeVault{}
	svc := NewService(nil, vault)

	note := Note{
		Title:   "My Resource",
		Content: "body",
		Type:    "resource",
		Kind:    "book",
	}
	_, err := svc.CreateOrUpdateNote(context.Background(), "/vault", note)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	fm := vault.lastNote.Frontmatter
	if fm["type"] != "resource" {
		t.Errorf("expected frontmatter type=resource, got %v", fm["type"])
	}
	if fm["kind"] != "book" {
		t.Errorf("expected frontmatter kind=book, got %v", fm["kind"])
	}
}

func TestService_CreateOrUpdateNote_DefaultsKindWhenOmitted(t *testing.T) {
	vault := &fakeVault{}
	svc := NewService(nil, vault)

	note := Note{
		Title:   "My Collection",
		Content: "body",
		Type:    "collection",
	}
	_, err := svc.CreateOrUpdateNote(context.Background(), "/vault", note)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	fm := vault.lastNote.Frontmatter
	if fm["kind"] != notemodel.ApplyDefaults(notemodel.TypeCollection, "", nil)["kind"] {
		t.Errorf("expected default kind for collection, got %v", fm["kind"])
	}
}

func TestService_CreateOrUpdateNote_NoApplyWhenTypeEmpty(t *testing.T) {
	vault := &fakeVault{}
	svc := NewService(nil, vault)

	note := Note{
		Title:   "Plain Note",
		Content: "body",
	}
	_, err := svc.CreateOrUpdateNote(context.Background(), "/vault", note)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vault.lastNote.Frontmatter != nil {
		t.Errorf("expected nil frontmatter for note without type, got %v", vault.lastNote.Frontmatter)
	}
}
