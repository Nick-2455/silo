package knowledge

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVaultStore_CreateOrUpdateNote(t *testing.T) {
	ctx := context.Background()
	vault := t.TempDir()
	store := VaultStore{}

	result, err := store.CreateOrUpdateNote(ctx, vault, Note{
		Title:   "Engram Memory",
		Content: "Engram is the source of truth.",
		Frontmatter: map[string]any{
			"source": "engram",
			"type":   "learning",
		},
	})
	if err != nil {
		t.Fatalf("create note: %v", err)
	}

	if !result.Created {
		t.Fatalf("expected first write to create note")
	}
	if !strings.HasPrefix(result.Path, filepath.Join(vault, "Silo", "Knowledge")) {
		t.Fatalf("note path escaped Silo knowledge dir: %s", result.Path)
	}

	data, err := os.ReadFile(result.Path)
	if err != nil {
		t.Fatalf("read note: %v", err)
	}
	text := string(data)
	for _, want := range []string{"---", "title: Engram Memory", "source: engram", "type: learning", "Engram is the source of truth."} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected note to contain %q, got:\n%s", want, text)
		}
	}

	updated, err := store.CreateOrUpdateNote(ctx, vault, Note{
		Title:   "Engram Memory",
		Content: "Updated content.",
	})
	if err != nil {
		t.Fatalf("update note: %v", err)
	}
	if updated.Created {
		t.Fatalf("expected second write to update existing note")
	}
}

func TestVaultStore_CreateOrUpdateNoteRejectsPathTraversal(t *testing.T) {
	store := VaultStore{}
	_, err := store.CreateOrUpdateNote(context.Background(), t.TempDir(), Note{
		Title:   "Bad",
		Path:    "../bad.md",
		Content: "should not write",
	})
	if err == nil {
		t.Fatalf("expected path traversal to be rejected")
	}
}

func TestVaultStore_SearchVault(t *testing.T) {
	ctx := context.Background()
	vault := t.TempDir()
	store := VaultStore{}

	if _, err := store.CreateOrUpdateNote(ctx, vault, Note{Title: "Agent Memory", Content: "Engram stores durable agent memory."}); err != nil {
		t.Fatalf("write matching note: %v", err)
	}
	if _, err := store.CreateOrUpdateNote(ctx, vault, Note{Title: "Cooking", Content: "Pasta notes."}); err != nil {
		t.Fatalf("write non-matching note: %v", err)
	}

	results, err := store.SearchVault(ctx, vault, "durable agent", 10)
	if err != nil {
		t.Fatalf("search vault: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Title != "Agent Memory" {
		t.Fatalf("unexpected result title: %q", results[0].Title)
	}
	if !strings.Contains(results[0].Snippet, "durable agent") {
		t.Fatalf("expected snippet to include query terms, got %q", results[0].Snippet)
	}
}
