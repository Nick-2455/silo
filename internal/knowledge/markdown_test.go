package knowledge

import (
	"strings"
	"testing"
)

func TestRenderMarkdown(t *testing.T) {
	note := Note{
		Title:   "Silo Bridge",
		Content: "Silo turns Engram memories into Markdown.",
		Frontmatter: map[string]any{
			"source":    "engram",
			"engram_id": "42",
		},
	}

	markdown, err := RenderMarkdown(note)
	if err != nil {
		t.Fatalf("render markdown: %v", err)
	}

	for _, want := range []string{
		"---",
		"title: Silo Bridge",
		"source: engram",
		"engram_id: \"42\"",
		"Silo turns Engram memories into Markdown.",
	} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("expected markdown to contain %q, got:\n%s", want, markdown)
		}
	}
}
