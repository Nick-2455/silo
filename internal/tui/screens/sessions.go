package screens

import (
	"fmt"
	"strings"

	"github.com/Nick-2455/marrow/internal/domain"
	"github.com/Nick-2455/marrow/internal/tui/styles"
)

// RenderSessionList renders a list of sessions grouped by project.
// cursor indicates the currently selected session index.
func RenderSessionList(sessions []domain.Session, cursor int) string {
	var b strings.Builder

	b.WriteString(styles.TitleStyle.Render("Sessions"))
	b.WriteString("\n\n")

	if len(sessions) == 0 {
		b.WriteString(styles.SubtextStyle.Render("No sessions yet. Use MCP tools to create sessions."))
		b.WriteString("\n")
		return b.String()
	}

	for i, s := range sessions {
		isSelected := i == cursor
		prefix := "  "
		if isSelected {
			prefix = styles.Cursor
		}

		line := fmt.Sprintf("%s %s", sessionIcon(), s.Description)
		if isSelected {
			b.WriteString(styles.SelectedStyle.Render(prefix + line))
		} else {
			b.WriteString(styles.UnselectedStyle.Render(prefix + line))
		}

		// Show project link
		if s.ProjectID != "" {
			b.WriteString(styles.SubtextStyle.Render(fmt.Sprintf("  [%s]", s.ProjectID)))
		}

		// Show date
		if !s.CreatedAt.IsZero() {
			b.WriteString(styles.SubtextStyle.Render(fmt.Sprintf("  %s", s.CreatedAt.Format("2006-01-02"))))
		}

		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(styles.HelpStyle.Render(fmt.Sprintf("%d session(s)  |  enter:details  esc:back", len(sessions))))
	b.WriteString("\n")

	return b.String()
}

// RenderSessionDetail renders a single session with its learnings.
func RenderSessionDetail(session domain.Session, learnings []domain.Learning, cursor int) string {
	var b strings.Builder

	b.WriteString(styles.TitleStyle.Render("Session: " + session.Description))
	b.WriteString("\n\n")

	// Project link
	b.WriteString(styles.HeadingStyle.Render("Project: "))
	b.WriteString(styles.SubtextStyle.Render(session.ProjectID))
	b.WriteString("\n\n")

	// Date
	if !session.CreatedAt.IsZero() {
		b.WriteString(styles.HeadingStyle.Render("Created: "))
		b.WriteString(styles.SubtextStyle.Render(session.CreatedAt.Format("2006-01-02 15:04")))
		b.WriteString("\n\n")
	}

	// Learnings from this session
	b.WriteString(styles.HeadingStyle.Render("Learnings:"))
	b.WriteString("\n")
	if len(learnings) == 0 {
		b.WriteString(styles.SubtextStyle.Render("  (none)"))
		b.WriteString("\n")
	} else {
		for i, l := range learnings {
			isSelected := i == cursor
			prefix := "  "
			if isSelected {
				prefix = styles.Cursor
			}
			preview := l.Content
			if len(preview) > 60 {
				preview = preview[:57] + "..."
			}
			label := fmt.Sprintf("• %s", preview)
			if isSelected {
				b.WriteString(styles.SelectedStyle.Render(prefix + label))
			} else {
				b.WriteString(styles.UnselectedStyle.Render(prefix + label))
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(styles.HelpStyle.Render(fmt.Sprintf("%d learning(s)  |  esc:back", len(learnings))))
	b.WriteString("\n")

	return b.String()
}

func sessionIcon() string {
	return "▸"
}
