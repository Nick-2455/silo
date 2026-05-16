package knowledge

// KnowledgeItem is a normalized knowledge record from Engram or another source.
type KnowledgeItem struct {
	ID       string         `json:"id,omitempty"`
	Title    string         `json:"title"`
	Type     string         `json:"type,omitempty"`
	Project  string         `json:"project,omitempty"`
	Content  string         `json:"content,omitempty"`
	Preview  string         `json:"preview,omitempty"`
	Source   string         `json:"source"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// Note is a Markdown note to create or update inside an Obsidian vault.
type Note struct {
	Title       string         `json:"title"`
	Path        string         `json:"path,omitempty"`
	Content     string         `json:"content"`
	Frontmatter map[string]any `json:"frontmatter,omitempty"`
	// Overwrite controls collision behavior when the resolved target path already exists.
	// Safe default is false (fail on collision).
	Overwrite bool `json:"overwrite,omitempty"`
	// Type is an optional community note type (concept, resource, roadmap, collection).
	// When non-empty, notemodel.ApplyDefaults is called to inject frontmatter defaults.
	Type string `json:"type,omitempty"`
	// Kind is an optional sub-category for notes that support it (e.g. resource kind: book).
	Kind string `json:"kind,omitempty"`
}

// NoteWriteResult describes a note write operation.
type NoteWriteResult struct {
	Path    string `json:"path"`
	Created bool   `json:"created"`
}

// NoteSearchResult is a lightweight match from Markdown vault search.
type NoteSearchResult struct {
	Title   string `json:"title"`
	Path    string `json:"path"`
	Snippet string `json:"snippet"`
}

// KnowledgeContext combines machine memory and human-readable notes.
type KnowledgeContext struct {
	Query  string             `json:"query"`
	Engram []KnowledgeItem    `json:"engram,omitempty"`
	Vault  []NoteSearchResult `json:"vault,omitempty"`
}
