package screens

import (
	"fmt"
	"strings"

	"github.com/Nick-2455/silo/internal/domain"
	"github.com/Nick-2455/silo/internal/tui/styles"
)

// RenderProjectList renders a list of projects with active/inactive indicators.
// cursor indicates the currently selected project index.
func RenderProjectList(projects []domain.Project, cursor int) string {
	var b strings.Builder

	b.WriteString(styles.TitleStyle.Render("Projects"))
	b.WriteString("\n\n")

	if len(projects) == 0 {
		b.WriteString(styles.SubtextStyle.Render("No projects yet. Use MCP tools to create projects."))
		b.WriteString("\n")
		return b.String()
	}

	for i, p := range projects {
		isSelected := i == cursor
		prefix := "  "
		if isSelected {
			prefix = styles.Cursor
		}

		// Active indicator: ▸ for active, • for inactive
		indicator := "▸"
		if !p.Active {
			indicator = "•"
		}

		line := fmt.Sprintf("%s %s", indicator, p.Name)
		if isSelected {
			b.WriteString(styles.SelectedStyle.Render(prefix + line))
		} else {
			b.WriteString(styles.UnselectedStyle.Render(prefix + line))
		}
		if !p.Active {
			b.WriteString(styles.SubtextStyle.Render(" (inactive)"))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(styles.HelpStyle.Render(fmt.Sprintf("%d project(s)  |  enter:details", len(projects))))
	b.WriteString("\n")

	return b.String()
}

// RenderProjectDetail renders a single project with its linked subareas.
func RenderProjectDetail(project domain.Project, subareas []domain.Subarea, cursor int) string {
	var b strings.Builder

	b.WriteString(styles.TitleStyle.Render(project.Name))
	b.WriteString("\n\n")

	// Status
	status := "Active"
	if !project.Active {
		status = "Inactive"
	}
	b.WriteString(styles.HeadingStyle.Render("Status: "))
	if project.Active {
		b.WriteString(styles.SuccessStyle.Render(status))
	} else {
		b.WriteString(styles.WarningStyle.Render(status))
	}
	b.WriteString("\n\n")

	// Description
	if project.Description != "" {
		b.WriteString(styles.HeadingStyle.Render("Description:"))
		b.WriteString("\n")
		b.WriteString(styles.UnselectedStyle.Render("  " + project.Description))
		b.WriteString("\n\n")
	}

	// Slug
	b.WriteString(styles.HeadingStyle.Render("Slug: "))
	b.WriteString(styles.SubtextStyle.Render(project.Slug))
	b.WriteString("\n\n")

	// Linked subareas
	b.WriteString(styles.HeadingStyle.Render("Linked Subareas:"))
	b.WriteString("\n")
	if len(subareas) == 0 {
		b.WriteString(styles.SubtextStyle.Render("  (none)"))
		b.WriteString("\n")
	} else {
		for i, s := range subareas {
			isSelected := i == cursor
			prefix := "  "
			if isSelected {
				prefix = styles.Cursor
			}
			label := s.Name
			if isSelected {
				b.WriteString(styles.SelectedStyle.Render(prefix + label))
			} else {
				b.WriteString(styles.UnselectedStyle.Render(prefix + label))
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(styles.HelpStyle.Render("esc:back"))
	b.WriteString("\n")

	return b.String()
}
