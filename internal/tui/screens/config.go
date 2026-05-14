package screens

import (
	"fmt"
	"strings"

	"github.com/Nick-2455/silo/internal/domain"
	"github.com/Nick-2455/silo/internal/tui/styles"
)

// RenderConfig renders the configuration screen as a read-only list.
func RenderConfig(config domain.Config, cursor int) string {
	var b strings.Builder

	b.WriteString(styles.TitleStyle.Render("Configuration"))
	b.WriteString("\n\n")

	fields := []struct {
		name  string
		value string
		mask  bool
	}{
		{"Profile", config.Profile, false},
		{"Engram Path", config.EngramPath, false},
		{"Triage Model", config.ModelPrefs.Triage, false},
		{"Summary Model", config.ModelPrefs.Summary, false},
	}

	for i, f := range fields {
		prefix := "  "
		if i == cursor {
			prefix = styles.Cursor
		}
		val := f.value
		if f.mask && val != "" {
			val = maskKey(val)
		}
		label := fmt.Sprintf("%s: %s", f.name, val)
		if i == cursor {
			b.WriteString(styles.SelectedStyle.Render(prefix + label))
		} else {
			b.WriteString(styles.UnselectedStyle.Render(prefix + label))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(styles.SubtextStyle.Render("(read-only)"))
	b.WriteString("\n")

	return b.String()
}

func maskKey(s string) string {
	if len(s) <= 8 {
		return "****"
	}
	return s[:4] + "****" + s[len(s)-4:]
}
