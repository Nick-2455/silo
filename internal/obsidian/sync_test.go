package obsidian

import (
	"strings"
	"testing"

	"github.com/Nick-2455/silo/internal/domain"
)

// TestBridgePath_ResourceNoteCarriesType verifies that a note written through the
// knowledge bridge path carries the community note type in its frontmatter.
// The obsidian.Syncer is NOT involved in this path — the test exercises the
// frontmatter struct directly to confirm the Kind field is wired correctly.
func TestFrontmatter_KindFieldPresent(t *testing.T) {
	syncer := &Syncer{VaultPath: t.TempDir()}

	fm := frontmatter{
		Type: "resource",
		Kind: "book",
		Slug: "my-resource",
	}
	output := syncer.marshalFrontmatter(fm)

	if !strings.Contains(output, "type: resource") {
		t.Errorf("expected type: resource in output, got:\n%s", output)
	}
	if !strings.Contains(output, "kind: book") {
		t.Errorf("expected kind: book in output, got:\n%s", output)
	}
}

// TestDomainExporter_Unchanged verifies that the legacy domain exporter
// still emits "type: domain" — the bridge path changes must not affect it.
func TestDomainExporter_Unchanged(t *testing.T) {
	syncer := &Syncer{VaultPath: t.TempDir()}

	node := domain.GraphNode{
		Title:    "Engineering",
		EngramID: "abc123",
		Active:   true,
	}
	output := syncer.buildDomainMarkdown(node, nil)

	if !strings.Contains(output, "type: domain") {
		t.Errorf("legacy domain exporter must still emit 'type: domain', got:\n%s", output)
	}
	if strings.Contains(output, "type: concept") ||
		strings.Contains(output, "type: resource") ||
		strings.Contains(output, "type: roadmap") ||
		strings.Contains(output, "type: collection") {
		t.Errorf("legacy domain exporter must not use community note types, got:\n%s", output)
	}
}

// TestSubareaExporter_Unchanged verifies that the legacy subarea exporter
// still emits "type: subarea".
func TestSubareaExporter_Unchanged(t *testing.T) {
	syncer := &Syncer{VaultPath: t.TempDir()}

	node := domain.GraphNode{Title: "Backend", EngramID: "sa1", Active: true}
	parent := domain.GraphNode{Title: "Engineering", EngramID: "d1", Active: true}
	output := syncer.buildSubareaMarkdown(node, parent)

	if !strings.Contains(output, "type: subarea") {
		t.Errorf("legacy subarea exporter must still emit 'type: subarea', got:\n%s", output)
	}
}
