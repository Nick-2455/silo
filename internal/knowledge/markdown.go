package knowledge

import (
	"bytes"
	"sort"

	"gopkg.in/yaml.v3"
)

// RenderMarkdown renders a note with deterministic YAML frontmatter.
func RenderMarkdown(note Note) (string, error) {
	frontmatter := make(map[string]any, len(note.Frontmatter)+1)
	for key, value := range note.Frontmatter {
		frontmatter[key] = value
	}
	frontmatter["title"] = note.Title

	var buf bytes.Buffer
	buf.WriteString("---\n")

	keys := make([]string, 0, len(frontmatter))
	for key := range frontmatter {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if idx := sort.SearchStrings(keys, "title"); idx < len(keys) && keys[idx] == "title" {
		keys = append([]string{"title"}, append(keys[:idx], keys[idx+1:]...)...)
	}

	for _, key := range keys {
		data, err := yaml.Marshal(map[string]any{key: frontmatter[key]})
		if err != nil {
			return "", err
		}
		buf.Write(data)
	}
	buf.WriteString("---\n\n")
	buf.WriteString(note.Content)
	if note.Content == "" || note.Content[len(note.Content)-1] != '\n' {
		buf.WriteByte('\n')
	}
	return buf.String(), nil
}
