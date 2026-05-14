package screens

import (
	"fmt"
	"strings"

	"github.com/Nick-2455/marrow/internal/tui/styles"
)

// RenderAdd renders the add resource form.
func RenderAdd(url string, title string, step int) string {
	var b strings.Builder

	b.WriteString(styles.TitleStyle.Render("Add Resource"))
	b.WriteString("\n\n")

	// URL field
	if step == 0 {
		b.WriteString(fmt.Sprintf("URL: %s▌\n", url))
	} else {
		b.WriteString(fmt.Sprintf("URL: %s\n", url))
	}

	// Title field (only after URL step)
	if step >= 1 {
		if step == 1 {
			b.WriteString(fmt.Sprintf("Title: %s▌\n", title))
		} else {
			b.WriteString(fmt.Sprintf("Title: %s\n", title))
		}
	}

	return b.String()
}
