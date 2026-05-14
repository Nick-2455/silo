package screens

import (
	"fmt"

	"github.com/Nick-2455/marrow/internal/tui/styles"
)

const maxTitleLen = 50

// Truncate shortens a string to max length, adding "..." if truncated.
func Truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

// RenderOptions renders a list of options with a cursor prefix on the selected item.
func RenderOptions(options []string, cursor int) string {
	var b string
	for idx, option := range options {
		if idx == cursor {
			b += styles.SelectedStyle.Render(styles.Cursor+option) + "\n"
		} else {
			b += styles.UnselectedStyle.Render("  "+option) + "\n"
		}
	}
	return b
}

// RenderBucketLine renders a bucket header with its count.
func RenderBucketLine(bucket string, count int, cursor int, idx int) string {
	prefix := "  "
	if idx == cursor {
		prefix = styles.Cursor
	}
	label := fmt.Sprintf("%s (%d)", bucket, count)
	if idx == cursor {
		return styles.SelectedStyle.Render(prefix + label)
	}
	return styles.UnselectedStyle.Render(prefix + label)
}

// RenderItemLine renders a single resource item line with optional cursor.
func RenderItemLine(title string, bucket string, cursor int, idx int) string {
	prefix := "  "
	if idx == cursor {
		prefix = styles.Cursor
	}
	label := fmt.Sprintf("%s [%s]", title, bucket)
	if idx == cursor {
		return styles.SelectedStyle.Render(prefix + label)
	}
	return styles.UnselectedStyle.Render(prefix + label)
}
