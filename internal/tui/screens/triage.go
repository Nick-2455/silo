package screens

import (
	"fmt"
	"strings"

	"github.com/Nick-2455/silo/internal/domain"
	"github.com/Nick-2455/silo/internal/tui/styles"
)

// RenderTriage renders the triage screen for moving a resource between buckets.
func RenderTriage(resource domain.Resource, bucketCursor int, moving bool, done bool, errStr string) string {
	var b strings.Builder

	if moving {
		b.WriteString(styles.HeadingStyle.Render("Moving resource..."))
		b.WriteString("\n")
		return b.String()
	}

	if done {
		b.WriteString(styles.SuccessStyle.Render(fmt.Sprintf("Moved to %s", resource.Bucket)))
		b.WriteString("\n")
		return b.String()
	}

	if errStr != "" {
		b.WriteString(styles.ErrorStyle.Render(fmt.Sprintf("Error: %s", errStr)))
		b.WriteString("\n\n")
	}

	title := resource.Title
	if title == "" {
		title = resource.URL
	}
	b.WriteString(styles.HeadingStyle.Render(fmt.Sprintf("Move: %s", Truncate(title, maxTitleLen))))
	b.WriteString("\n\n")

	buckets := domain.AllBuckets()
	for i, bucket := range buckets {
		prefix := "  "
		if i == bucketCursor {
			prefix = styles.Cursor
		}
		if i == bucketCursor {
			b.WriteString(styles.SelectedStyle.Render(prefix + string(bucket)))
		} else {
			b.WriteString(styles.UnselectedStyle.Render(prefix + string(bucket)))
		}
		b.WriteString("\n")
	}

	return b.String()
}
