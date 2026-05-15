package screens

import (
	"fmt"
	"strings"

	"github.com/Nick-2455/silo/internal/domain"
	"github.com/Nick-2455/silo/internal/tui/styles"
)

// RenderLearningList renders a list of learnings with content preview and linked subareas.
// cursor indicates the currently selected learning index.
func RenderLearningList(learnings []domain.Learning, cursor int) string {
	var b strings.Builder

	b.WriteString(styles.TitleStyle.Render("Learnings"))
	b.WriteString("\n\n")

	if len(learnings) == 0 {
		b.WriteString(styles.SubtextStyle.Render("No learnings yet. Use MCP tools to create learnings."))
		b.WriteString("\n")
		return b.String()
	}

	for i, l := range learnings {
		isSelected := i == cursor
		prefix := "  "
		if isSelected {
			prefix = styles.Cursor
		}

		// Content preview
		preview := l.Content
		if len(preview) > 70 {
			preview = preview[:67] + "..."
		}
		line := fmt.Sprintf("%s %s", learningIcon(), preview)
		if isSelected {
			b.WriteString(styles.SelectedStyle.Render(prefix + line))
		} else {
			b.WriteString(styles.UnselectedStyle.Render(prefix + line))
		}
		b.WriteString("\n")

		// Linked subareas
		if len(l.SubareaIDs) > 0 {
			subLabels := make([]string, 0, len(l.SubareaIDs))
			for _, sid := range l.SubareaIDs {
				subLabels = append(subLabels, shortenID(sid))
			}
			b.WriteString(styles.SubtextStyle.Render(fmt.Sprintf("    subareas: %s", strings.Join(subLabels, ", "))))
			b.WriteString("\n")
		}

		// Date
		if !l.CreatedAt.IsZero() {
			b.WriteString(styles.SubtextStyle.Render(fmt.Sprintf("    %s", l.CreatedAt.Format("2006-01-02"))))
			b.WriteString("\n")
		}

		b.WriteString("\n")
	}

	b.WriteString(styles.HelpStyle.Render(fmt.Sprintf("%d learning(s)  |  esc:back", len(learnings))))
	b.WriteString("\n")

	return b.String()
}

func learningIcon() string {
	return "◆"
}

// shortenID extracts the last segment of an ID path for display.
// "subarea/dev/backend" → "backend"
func shortenID(id string) string {
	if idx := strings.LastIndex(id, "/"); idx >= 0 && idx+1 < len(id) {
		return id[idx+1:]
	}
	return id
}
