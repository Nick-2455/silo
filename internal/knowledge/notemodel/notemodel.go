// Package notemodel defines the community-facing note vocabulary used by
// Silo's Markdown bridge and MCP tool surface.
//
// Four base types (concept, resource, roadmap, collection) with optional
// kind discriminators replace ad-hoc note framing. Templates ship as
// embedded Markdown files and are merged into note frontmatter at creation
// time.
package notemodel

import (
	"embed"
	"fmt"
	"strings"
)

// Type is a base note category.
type Type string

const (
	TypeConcept    Type = "concept"
	TypeResource   Type = "resource"
	TypeRoadmap    Type = "roadmap"
	TypeCollection Type = "collection"
)

// validTypes is the authoritative set of supported base types.
var validTypes = map[Type]bool{
	TypeConcept:    true,
	TypeResource:   true,
	TypeRoadmap:    true,
	TypeCollection: true,
}

// collectionKinds lists supported kind values for collection notes.
var collectionKinds = map[string]bool{
	"subject":     true,
	"project":     true,
	"interest":    true,
	"career-path": true,
	"research":    true,
	"team-space":  true,
	"other":       true,
}

// resourceKinds lists supported kind values for resource notes.
var resourceKinds = map[string]bool{
	"article": true,
	"book":    true,
	"video":   true,
	"tool":    true,
	"paper":   true,
	"course":  true,
	"other":   true,
}

// defaultKinds maps a type to the kind it defaults to when kind is omitted.
var defaultKinds = map[Type]string{
	TypeCollection: "other",
	TypeResource:   "other",
}

// kindSets maps types to their allowed kind sets (nil → no kind restriction).
var kindSets = map[Type]map[string]bool{
	TypeCollection: collectionKinds,
	TypeResource:   resourceKinds,
}

// FrontmatterSchema describes the frontmatter fields for a note type.
type FrontmatterSchema struct {
	Required []string `json:"required"`
	Optional []string `json:"optional,omitempty"`
}

// Template represents a note template with its metadata and embedded body.
type Template struct {
	Type              Type              `json:"type"`
	DefaultKind       string            `json:"default_kind,omitempty"`
	Description       string            `json:"description"`
	FrontmatterSchema FrontmatterSchema `json:"frontmatter_schema"`
	body              string
}

// Body returns the raw Markdown body of the template.
func (t Template) Body() string { return t.body }

//go:embed templates
var templateFS embed.FS

var catalog []Template

func init() {
	catalog = []Template{
		{
			Type:        TypeConcept,
			Description: "A discrete idea, term, or principle worth capturing.",
			FrontmatterSchema: FrontmatterSchema{
				Required: []string{"type", "title", "tags", "created"},
			},
		},
		{
			Type:        TypeResource,
			DefaultKind: "other",
			Description: "An external resource (article, book, video, tool, paper, course).",
			FrontmatterSchema: FrontmatterSchema{
				Required: []string{"type", "kind", "title", "tags", "created"},
				Optional: []string{"url", "source", "status"},
			},
		},
		{
			Type:        TypeRoadmap,
			Description: "A staged learning or project plan with status tracking.",
			FrontmatterSchema: FrontmatterSchema{
				Required: []string{"type", "title", "tags", "status", "created"},
				Optional: []string{"stages"},
			},
		},
		{
			Type:        TypeCollection,
			DefaultKind: "other",
			Description: "A named grouping of related notes (subject, project, interest, career-path, research, team-space).",
			FrontmatterSchema: FrontmatterSchema{
				Required: []string{"type", "kind", "title", "tags", "created"},
				Optional: []string{"items"},
			},
		},
	}

	for i, tmpl := range catalog {
		path := fmt.Sprintf("templates/%s.md", tmpl.Type)
		data, err := templateFS.ReadFile(path)
		if err != nil {
			// This should never happen — embedded files are baked at compile time.
			panic(fmt.Sprintf("notemodel: missing embedded template for %q: %v", tmpl.Type, err))
		}
		catalog[i].body = string(data)
	}
}

// Templates returns a copy of all registered templates (metadata only, no body).
// Use Get to retrieve the body for a specific type.
func Templates() []Template {
	out := make([]Template, len(catalog))
	copy(out, catalog)
	return out
}

// Get returns the template for the given type, or false if unknown.
func Get(t Type) (Template, bool) {
	for _, tmpl := range catalog {
		if tmpl.Type == t {
			return tmpl, true
		}
	}
	return Template{}, false
}

// ValidType reports whether t is a recognized base type.
func ValidType(t Type) bool {
	return validTypes[t]
}

// ValidKind reports whether kind is valid for the given type.
// Types with no kind set accept any kind value.
// An empty kind string is always considered valid (will be defaulted).
func ValidKind(t Type, kind string) bool {
	if kind == "" {
		return true
	}
	set, ok := kindSets[t]
	if !ok {
		// No restriction on this type.
		return true
	}
	return set[kind]
}

// ApplyDefaults merges note-model defaults into fm for the given type and kind.
// It is a pure function — fm is not mutated; a new map is returned.
//
// Rules:
//   - "type" is always set from t (even if t is unknown).
//   - If kind is non-empty and the type supports a kind, it is stored under "kind".
//   - If kind is empty and the type has a default kind, the default is used.
//   - Unknown type values are accepted (soft validation — community forward compat).
func ApplyDefaults(t Type, kind string, fm map[string]any) map[string]any {
	out := make(map[string]any, len(fm)+4)
	for k, v := range fm {
		out[k] = v
	}

	out["type"] = string(t)

	resolvedKind := kind
	if resolvedKind == "" {
		resolvedKind = defaultKinds[t]
	}
	if resolvedKind != "" {
		out["kind"] = resolvedKind
	}

	return out
}

// AllowedKinds returns the list of allowed kind values for t, or nil if t has
// no kind restriction.
func AllowedKinds(t Type) []string {
	set, ok := kindSets[t]
	if !ok {
		return nil
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	return out
}

// TypeNames returns the names of all supported base types.
func TypeNames() []string {
	names := make([]string, 0, len(validTypes))
	for t := range validTypes {
		names = append(names, string(t))
	}
	return names
}

// Validate checks whether t is a known type and kind is acceptable for it.
// Returns an error describing valid options when validation fails.
func Validate(t Type, kind string) error {
	if !ValidType(t) {
		return fmt.Errorf("unknown note type %q: valid types are %s",
			t, strings.Join(TypeNames(), ", "))
	}
	if !ValidKind(t, kind) {
		return fmt.Errorf("unknown kind %q for type %q: valid kinds are %s",
			kind, t, strings.Join(AllowedKinds(t), ", "))
	}
	return nil
}
