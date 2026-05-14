package screens

import (
	"fmt"
	"strings"

	"github.com/Nick-2455/marrow/internal/domain"
	"github.com/Nick-2455/marrow/internal/tui/styles"
)

// RenderDomainTree renders a hierarchical tree of domains and their subareas.
// cursor indicates the currently selected domain index.
func RenderDomainTree(domains []domain.DomainWithSubareas, cursor int) string {
	var b strings.Builder

	b.WriteString(styles.TitleStyle.Render("Taxonomy — Domain Tree"))
	b.WriteString("\n\n")

	if len(domains) == 0 {
		b.WriteString(styles.SubtextStyle.Render("No domains yet. Use MCP tools to create domains and subareas."))
		b.WriteString("\n")
		return b.String()
	}

	for i, d := range domains {
		isSelected := i == cursor
		prefix := "  "
		if isSelected {
			prefix = styles.Cursor
		}

		// Domain line
		domainLabel := fmt.Sprintf("%s %s", domainIcon(d.Domain.Active), d.Domain.Title)
		if isSelected {
			b.WriteString(styles.SelectedStyle.Render(prefix + domainLabel))
		} else {
			b.WriteString(styles.UnselectedStyle.Render(prefix + domainLabel))
		}
		if !d.Domain.Active {
			b.WriteString(styles.SubtextStyle.Render(" (inactive)"))
		}
		b.WriteString("\n")

		// Subareas (indented)
		for _, s := range d.Subareas {
			subPrefix := "    • "
			if isSelected {
				subPrefix = "      "
			}
			subLabel := s.Title
			if !s.Active {
				subLabel += " (inactive)"
			}
			b.WriteString(styles.SubtextStyle.Render(subPrefix + subLabel))
			b.WriteString("\n")
		}

		if len(d.Subareas) == 0 {
			b.WriteString(styles.SubtextStyle.Render("    (no subareas)"))
			b.WriteString("\n")
		}

		b.WriteString("\n")
	}

	b.WriteString(styles.HelpStyle.Render(fmt.Sprintf("%d domain(s)", len(domains))))
	b.WriteString("\n")

	return b.String()
}

func domainIcon(active bool) string {
	if active {
		return "▸"
	}
	return "•"
}
