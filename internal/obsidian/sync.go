package obsidian

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/Nick-2455/silo/internal/domain"
)

// Syncer exports the Silo knowledge graph as markdown files
// with YAML frontmatter and wikilinks for Obsidian.
type Syncer struct {
	Store     domain.GraphStore
	VaultPath string
}

// SyncReport summarizes the result of a sync operation.
type SyncReport struct {
	NodesWritten int      `json:"nodes_written"`
	EdgesWritten int      `json:"edges_written"`
	Errors       []string `json:"errors,omitempty"`
}

// SyncAll exports the complete graph as markdown files under VaultPath/Silo/.
func (s *Syncer) SyncAll(ctx context.Context) (*SyncReport, error) {
	if s.VaultPath == "" {
		return nil, fmt.Errorf("no vault path configured")
	}

	report := &SyncReport{}

	// Create directory structure
	base := filepath.Join(s.VaultPath, "Silo")
	dirs := []string{
		filepath.Join(base, "Domains"),
		filepath.Join(base, "Subareas"),
		filepath.Join(base, "Projects"),
		filepath.Join(base, "Sessions"),
		filepath.Join(base, "Learnings"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return nil, fmt.Errorf("create dir %s: %w", d, err)
		}
	}

	// Build a title→engramID map for deduplication
	titleMap := make(map[string]string) // title → first engramID seen

	// 1. Domains
	tree, err := s.Store.GetDomainTree(ctx)
	if err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("get domain tree: %v", err))
	} else {
		for _, dws := range tree {
			dn := dws.Domain
			titleMap[dn.Title] = dn.EngramID

			// Gather subarea wikilinks
			subLinks := make([]string, 0, len(dws.Subareas))
			for _, sa := range dws.Subareas {
				subLinks = append(subLinks, fmt.Sprintf("[[%s]]", sa.Title))
				titleMap[sa.Title] = sa.EngramID
			}

			content := s.buildDomainMarkdown(dn, subLinks)
			path := s.filePath("Domains", dn.Title)
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				report.Errors = append(report.Errors, fmt.Sprintf("write %s: %v", path, err))
				continue
			}
			report.NodesWritten++

			// Write subarea files
			for _, sa := range dws.Subareas {
				// Skip if title collision (already written)
				if titleMap[sa.Title] != sa.EngramID {
					continue
				}
				subContent := s.buildSubareaMarkdown(sa, dn)
				subPath := s.filePath("Subareas", sa.Title)
				if err := os.WriteFile(subPath, []byte(subContent), 0o644); err != nil {
					report.Errors = append(report.Errors, fmt.Sprintf("write %s: %v", subPath, err))
					continue
				}
				report.NodesWritten++
			}
		}
	}

	// 2. Projects
	projects, err := s.Store.ListNodesByType(ctx, domain.NodeTypeProject)
	if err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("list projects: %v", err))
	} else {
		for _, p := range projects {
			if titleMap[p.Title] != "" {
				// title collision — skip
				continue
			}
			titleMap[p.Title] = p.EngramID

			// Gather subarea links via applies_to edges
			edges, err := s.Store.GetEdges(ctx, p.EngramID, "from")
			if err != nil {
				report.Errors = append(report.Errors, fmt.Sprintf("get edges for project %s: %v", p.Title, err))
				continue
			}

			subLinks := make([]string, 0)
			for _, e := range edges {
				if e.Label == domain.EdgeAppliesTo {
					subNode, err := s.Store.GetNode(ctx, e.ToID)
					if err == nil {
						subLinks = append(subLinks, fmt.Sprintf("[[%s]]", subNode.Title))
						report.EdgesWritten++
					}
				}
			}

			content := s.buildProjectMarkdown(p, subLinks)
			path := s.filePath("Projects", p.Title)
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				report.Errors = append(report.Errors, fmt.Sprintf("write %s: %v", path, err))
				continue
			}
			report.NodesWritten++
		}
	}

	// 3. Sessions
	sessions, err := s.Store.ListNodesByType(ctx, domain.NodeTypeSession)
	if err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("list sessions: %v", err))
	} else {
		for _, sess := range sessions {
			if titleMap[sess.Title] != "" {
				continue
			}
			titleMap[sess.Title] = sess.EngramID

			// Find linked project via worked_on edge (session → project is "to" direction)
			edges, err := s.Store.GetEdges(ctx, sess.EngramID, "from")
			if err != nil {
				report.Errors = append(report.Errors, fmt.Sprintf("get edges for session %s: %v", sess.Title, err))
				continue
			}

			var projectLink string
			for _, e := range edges {
				if e.Label == domain.EdgeWorkedOn {
					projNode, err := s.Store.GetNode(ctx, e.ToID)
					if err == nil {
						projectLink = projNode.Title
						report.EdgesWritten++
					}
					break
				}
			}

			content := s.buildSessionMarkdown(sess, projectLink)
			path := s.filePath("Sessions", sess.Title)
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				report.Errors = append(report.Errors, fmt.Sprintf("write %s: %v", path, err))
				continue
			}
			report.NodesWritten++
		}
	}

	// 4. Learnings
	learnings, err := s.Store.ListLearnings(ctx, "")
	if err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("list learnings: %v", err))
	} else {
		for _, l := range learnings {
			if titleMap[l.Content] != "" {
				continue
			}
			titleMap[l.Content] = l.ID

			// Gather subarea and session links
			subLinks := make([]string, 0)
			for _, sid := range l.SubareaIDs {
				subNode, err := s.Store.GetNode(ctx, sid)
				if err == nil {
					subLinks = append(subLinks, fmt.Sprintf("[[%s]]", subNode.Title))
					report.EdgesWritten++
				}
			}

			var sessionLink string
			if l.SessionID != "" {
				sessNode, err := s.Store.GetNode(ctx, l.SessionID)
				if err == nil {
					sessionLink = sessNode.Title
					report.EdgesWritten++
				}
			}

			content := s.buildLearningMarkdown(l, subLinks, sessionLink)
			path := s.filePath("Learnings", l.Content)
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				report.Errors = append(report.Errors, fmt.Sprintf("write %s: %v", path, err))
				continue
			}
			report.NodesWritten++
		}
	}

	// 5. Person
	persons, err := s.Store.ListNodesByType(ctx, domain.NodeTypePerson)
	if err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("list persons: %v", err))
	} else {
		for _, p := range persons {
			if !p.Active {
				continue
			}
			if titleMap[p.Title] != "" {
				continue
			}
			titleMap[p.Title] = p.EngramID

			content := s.buildPersonMarkdown(p)
			path := filepath.Join(base, "Persona.md")
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				report.Errors = append(report.Errors, fmt.Sprintf("write %s: %v", path, err))
				continue
			}
			report.NodesWritten++
		}
	}

	return report, nil
}

// sanitizeFilename replaces characters that are invalid in filenames.
func sanitizeFilename(name string) string {
	r := strings.NewReplacer("/", "-", "\\", "-", ":", "-")
	return r.Replace(name)
}

// filePath returns the full path for a markdown file in a subdirectory.
func (s *Syncer) filePath(subdir, title string) string {
	name := sanitizeFilename(title)
	return filepath.Join(s.VaultPath, "Silo", subdir, name+".md")
}

// --- Markdown builders ---

type frontmatter struct {
	Type     string   `yaml:"type"`
	Kind     string   `yaml:"kind,omitempty"`
	Slug     string   `yaml:"slug,omitempty"`
	Active   bool     `yaml:"active,omitempty"`
	Subareas []string `yaml:"subareas,omitempty"`
}

func (s *Syncer) marshalFrontmatter(fm frontmatter) string {
	data, err := yaml.Marshal(&fm)
	if err != nil {
		return "---\n---\n"
	}
	return "---\n" + string(data) + "---\n"
}

func (s *Syncer) buildDomainMarkdown(node domain.GraphNode, subLinks []string) string {
	fm := frontmatter{
		Type:     "domain",
		Slug:     domain.Slugify(node.Title),
		Active:   node.Active,
		Subareas: subLinks,
	}
	return s.marshalFrontmatter(fm) + "\n# " + node.Title + "\n\nDomain node.\n"
}

func (s *Syncer) buildSubareaMarkdown(node domain.GraphNode, parent domain.GraphNode) string {
	fm := frontmatter{
		Type:   "subarea",
		Slug:   domain.Slugify(node.Title),
		Active: node.Active,
	}
	content := s.marshalFrontmatter(fm) + "\n# " + node.Title + "\n\n"
	content += "Subarea of [[" + parent.Title + "]].\n"
	return content
}

func (s *Syncer) buildProjectMarkdown(node domain.GraphNode, subLinks []string) string {
	fm := frontmatter{
		Type:     "project",
		Slug:     domain.Slugify(node.Title),
		Active:   node.Active,
		Subareas: subLinks,
	}
	return s.marshalFrontmatter(fm) + "\n# " + node.Title + "\n\nProject node.\n"
}

func (s *Syncer) buildSessionMarkdown(node domain.GraphNode, projectTitle string) string {
	fm := frontmatter{
		Type: "session",
		Slug: domain.Slugify(node.Title),
	}
	content := s.marshalFrontmatter(fm) + "\n# " + node.Title + "\n\n"
	if projectTitle != "" {
		content += "Session from [[" + projectTitle + "]].\n"
	} else {
		content += "Session node.\n"
	}
	return content
}

func (s *Syncer) buildLearningMarkdown(l domain.Learning, subLinks []string, sessionTitle string) string {
	fm := map[string]any{
		"type":   "learning",
		"active": true,
	}
	if len(subLinks) > 0 {
		fm["subareas"] = subLinks
	}

	data, err := yaml.Marshal(&fm)
	if err != nil {
		data = []byte("---\n---\n")
	}

	content := "---\n" + string(data) + "---\n"
	content += "\n# Learning\n\n" + l.Content + "\n"
	if sessionTitle != "" {
		content += "\nFrom [[" + sessionTitle + "]].\n"
	}
	return content
}

func (s *Syncer) buildPersonMarkdown(node domain.GraphNode) string {
	fm := map[string]any{
		"type":   "person",
		"active": node.Active,
	}
	data, err := yaml.Marshal(&fm)
	if err != nil {
		data = []byte("---\n---\n")
	}
	return "---\n" + string(data) + "---\n\n# " + node.Title + "\n\nPerson node.\n"
}
