package knowledge

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

const defaultKnowledgeDir = "Silo/Knowledge"

// VaultStore performs simple Markdown operations inside an Obsidian vault.
type VaultStore struct{}

// CreateOrUpdateNote writes a Markdown note under the vault's Silo namespace.
func (s VaultStore) CreateOrUpdateNote(ctx context.Context, vaultPath string, note Note) (NoteWriteResult, error) {
	if err := ctx.Err(); err != nil {
		return NoteWriteResult{}, err
	}
	if vaultPath == "" {
		return NoteWriteResult{}, errors.New("vault path is required")
	}
	if strings.TrimSpace(note.Title) == "" {
		return NoteWriteResult{}, errors.New("note title is required")
	}

	target, err := resolveNotePath(vaultPath, note)
	if err != nil {
		return NoteWriteResult{}, err
	}

	markdown, err := RenderMarkdown(note)
	if err != nil {
		return NoteWriteResult{}, fmt.Errorf("render markdown: %w", err)
	}

	exists := false
	if _, err := os.Stat(target); errors.Is(err, os.ErrNotExist) {
		exists = false
	} else if err != nil {
		return NoteWriteResult{}, fmt.Errorf("stat note: %w", err)
	} else {
		exists = true
	}
	if exists && !note.Overwrite {
		return NoteWriteResult{}, fmt.Errorf("note already exists: %s", target)
	}
	created := !exists

	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return NoteWriteResult{}, fmt.Errorf("create note dir: %w", err)
	}
	if err := os.WriteFile(target, []byte(markdown), 0o644); err != nil {
		return NoteWriteResult{}, fmt.Errorf("write note: %w", err)
	}

	return NoteWriteResult{Path: target, Created: created}, nil
}

// SearchVault scans Markdown files in a vault for a case-insensitive query.
func (s VaultStore) SearchVault(ctx context.Context, vaultPath, query string, limit int) ([]NoteSearchResult, error) {
	if vaultPath == "" {
		return nil, errors.New("vault path is required")
	}
	if limit <= 0 {
		limit = 20
	}
	if info, err := os.Stat(vaultPath); err != nil {
		return nil, fmt.Errorf("vault path: %w", err)
	} else if !info.IsDir() {
		return nil, errors.New("vault path must be a directory")
	}

	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return nil, errors.New("query is required")
	}

	results := make([]NoteSearchResult, 0, limit)
	err := filepath.WalkDir(vaultPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".md" || len(results) >= limit {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		text := string(data)
		idx := strings.Index(strings.ToLower(text), query)
		if idx == -1 {
			return nil
		}

		results = append(results, NoteSearchResult{
			Title:   titleFromMarkdown(path, text),
			Path:    path,
			Snippet: snippet(text, idx, len(query)),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return results, nil
}

func resolveNotePath(vaultPath string, note Note) (string, error) {
	vaultAbs, err := filepath.Abs(vaultPath)
	if err != nil {
		return "", fmt.Errorf("resolve vault path: %w", err)
	}

	rel := note.Path
	if strings.TrimSpace(rel) == "" {
		rel = filepath.Join(defaultKnowledgeDir, sanitizeFilename(note.Title)+".md")
	}
	if filepath.IsAbs(rel) {
		return "", errors.New("note path must be relative")
	}
	if filepath.Ext(rel) != ".md" {
		rel += ".md"
	}

	target := filepath.Clean(filepath.Join(vaultAbs, rel))
	inside, err := isInside(vaultAbs, target)
	if err != nil {
		return "", err
	}
	if !inside {
		return "", errors.New("note path escapes vault")
	}

	// Compatibility: older versions derived default filenames from Title without lowercasing.
	// On case-sensitive filesystems that can create duplicates (e.g. Engram-Memory.md vs engram-memory.md).
	// For title-derived writes with overwrite=true, prefer updating an existing legacy-cased file.
	if strings.TrimSpace(note.Path) == "" && note.Overwrite {
		if legacy, ok := findExistingCaseInsensitive(target); ok {
			target = legacy
		}
	}
	return target, nil
}

func findExistingCaseInsensitive(target string) (string, bool) {
	if _, err := os.Stat(target); err == nil {
		return target, true
	}

	dir := filepath.Dir(target)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", false
	}
	want := strings.ToLower(filepath.Base(target))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.ToLower(e.Name()) == want {
			return filepath.Join(dir, e.Name()), true
		}
	}
	return "", false
}

func isInside(root, target string) (bool, error) {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return false, err
	}
	return rel == "." || (!strings.HasPrefix(rel, "..") && !filepath.IsAbs(rel)), nil
}

func sanitizeFilename(name string) string {
	name = strings.TrimSpace(name)
	var b strings.Builder
	lastDash := false
	for _, r := range name {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			lastDash = false
		case r == '-' || r == '_' || unicode.IsSpace(r):
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	clean := strings.ToLower(strings.Trim(b.String(), "-"))
	if clean == "" {
		return "note"
	}
	return clean
}

func titleFromMarkdown(path, text string) string {
	for _, line := range strings.Split(text, "\n") {
		if title, ok := strings.CutPrefix(line, "title: "); ok {
			return strings.Trim(strings.TrimSpace(title), "\"")
		}
	}
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

func snippet(text string, idx, queryLen int) string {
	start := idx - 60
	if start < 0 {
		start = 0
	}
	end := idx + queryLen + 80
	if end > len(text) {
		end = len(text)
	}
	return strings.TrimSpace(text[start:end])
}
