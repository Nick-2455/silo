package screens

import (
	"fmt"
	"strings"

	"github.com/Nick-2455/marrow/internal/domain"
	"github.com/Nick-2455/marrow/internal/tui/styles"
)

const maxDashItems = 5

// RenderDashboard renders the dashboard view showing resources grouped by bucket.
func RenderDashboard(resources []domain.Resource, cursor int) string {
	var b strings.Builder

	b.WriteString(styles.TitleStyle.Render("marrow — Dashboard"))
	b.WriteString("\n\n")

	// Group by bucket
	buckets := make(map[domain.Bucket][]domain.Resource)
	for _, r := range resources {
		buckets[r.Bucket] = append(buckets[r.Bucket], r)
	}

	for i, bucket := range domain.AllBuckets() {
		items := buckets[bucket]
		count := len(items)

		b.WriteString(RenderBucketLine(string(bucket), count, cursor, i))
		b.WriteString("\n")

		if count == 0 {
			b.WriteString(styles.SubtextStyle.Render("  (empty)"))
			b.WriteString("\n")
		} else {
			limit := maxDashItems
			if count < limit {
				limit = count
			}
			for i := 0; i < limit; i++ {
				title := items[i].Title
				if title == "" {
					title = items[i].URL
				}
				b.WriteString(fmt.Sprintf("  • %s\n", Truncate(title, maxTitleLen)))
			}
			if count > maxDashItems {
				b.WriteString(styles.SubtextStyle.Render(fmt.Sprintf("  ... and %d more", count-maxDashItems)))
				b.WriteString("\n")
			}
		}
		b.WriteString("\n")
	}

	return b.String()
}
