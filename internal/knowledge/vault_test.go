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
		Overwrite: true,
	})
	if err != nil {
		t.Fatalf("update note: %v", err)
	}
	if updated.Created {
		t.Fatalf("expected second write to update existing note")
	}
}

func TestVaultStore_CreateOrUpdateNote_CollisionRequiresOverwrite(t *testing.T) {
	ctx := context.Background()
	vault := t.TempDir()
	store := VaultStore{}

	first, err := store.CreateOrUpdateNote(ctx, vault, Note{Title: "Intro to Type Theory", Content: "first"})
	if err != nil {
		t.Fatalf("first write: %v", err)
	}
	if !first.Created {
		t.Fatalf("expected first write to create")
	}
	if got := filepath.Base(first.Path); got != "intro-to-type-theory.md" {
		t.Fatalf("expected lowercase slug filename, got %q", got)
	}

	if _, err := store.CreateOrUpdateNote(ctx, vault, Note{Title: "Intro to Type Theory", Content: "second"}); err == nil {
		t.Fatalf("expected collision error when overwrite=false")
	}

	overwritten, err := store.CreateOrUpdateNote(ctx, vault, Note{Title: "Intro to Type Theory", Content: "third", Overwrite: true})
	if err != nil {
		t.Fatalf("overwrite write: %v", err)
	}
	if overwritten.Created {
		t.Fatalf("expected overwrite write to report Created=false")
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

func TestVaultStore_CreateOrUpdateNote_ExplicitPathCollisionRequiresOverwrite(t *testing.T) {
	ctx := context.Background()
	vault := t.TempDir()
	store := VaultStore{}

	rel := filepath.Join("Silo", "Knowledge", "custom.md")
	first, err := store.CreateOrUpdateNote(ctx, vault, Note{Title: "A", Path: rel, Content: "first"})
	if err != nil {
		t.Fatalf("first write: %v", err)
	}
	if !first.Created {
		t.Fatalf("expected first write to create")
	}

	if _, err := store.CreateOrUpdateNote(ctx, vault, Note{Title: "B", Path: rel, Content: "second"}); err == nil {
		t.Fatalf("expected collision error when overwrite=false")
	}

	overwritten, err := store.CreateOrUpdateNote(ctx, vault, Note{Title: "B", Path: rel, Content: "third", Overwrite: true})
	if err != nil {
		t.Fatalf("overwrite write: %v", err)
	}
	if overwritten.Created {
		t.Fatalf("expected overwrite write to report Created=false")
	}
}

func TestVaultStore_CreateOrUpdateNote_OverwriteUpdatesLegacyCasedDefaultPath(t *testing.T) {
	ctx := context.Background()
	vault := t.TempDir()
	store := VaultStore{}

	legacyRel := filepath.Join("Silo", "Knowledge", "Engram-Memory.md")
	legacyAbs := filepath.Join(vault, legacyRel)
	if err := os.MkdirAll(filepath.Dir(legacyAbs), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(legacyAbs, []byte("old"), 0o644); err != nil {
		t.Fatalf("seed legacy note: %v", err)
	}

	res, err := store.CreateOrUpdateNote(ctx, vault, Note{Title: "Engram Memory", Content: "new", Overwrite: true})
	if err != nil {
		t.Fatalf("overwrite update: %v", err)
	}
	// On case-sensitive filesystems we expect targeting the legacy-cased file.
	// On macOS default (case-insensitive) both paths may refer to the same inode.
	legacyInfo, err := os.Stat(legacyAbs)
	if err != nil {
		t.Fatalf("stat legacy note: %v", err)
	}
	resInfo, err := os.Stat(res.Path)
	if err != nil {
		t.Fatalf("stat result note: %v", err)
	}
	if !os.SameFile(legacyInfo, resInfo) {
		t.Fatalf("expected update to target legacy file, got different path %s", res.Path)
	}
	if res.Created {
		t.Fatalf("expected overwrite update to report Created=false")
	}

	data, err := os.ReadFile(legacyAbs)
	if err != nil {
		t.Fatalf("read legacy note: %v", err)
	}
	if !strings.Contains(string(data), "new") {
		t.Fatalf("expected legacy note content to be updated, got: %s", string(data))
	}

	// Ensure we did not create a second file.
	entries, err := os.ReadDir(filepath.Join(vault, defaultKnowledgeDir))
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	mdCount := 0
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			mdCount++
		}
	}
	if mdCount != 1 {
		t.Fatalf("expected exactly 1 markdown file in %s, got %d", defaultKnowledgeDir, mdCount)
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
