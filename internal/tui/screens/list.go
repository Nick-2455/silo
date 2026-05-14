package screens

import (
	"fmt"
	"strings"

	"github.com/Nick-2455/marrow/internal/domain"
	"github.com/Nick-2455/marrow/internal/tui/styles"
)

// RenderList renders the list view with optional filter and search.
func RenderList(resources []domain.Resource, bucketFilter domain.Bucket, searchQuery string, searching bool, cursor int) string {
	var b strings.Builder

	filtered := FilteredResources(resources, bucketFilter, searchQuery)

	if searching {
		b.WriteString(styles.TitleStyle.Render(fmt.Sprintf("/ %s", searchQuery)))
		b.WriteString("\n\n")
	}

	if bucketFilter != "" {
		b.WriteString(styles.HeadingStyle.Render(fmt.Sprintf("Filter: %s", bucketFilter)))
		b.WriteString("\n\n")
	}

	if len(filtered) == 0 {
		b.WriteString(styles.SubtextStyle.Render("No resources found"))
		b.WriteString("\n")
		return b.String()
	}

	for i, r := range filtered {
		title := Truncate(r.Title, maxTitleLen)
		if title == "" {
			title = Truncate(r.URL, maxTitleLen)
		}
		b.WriteString(RenderItemLine(title, string(r.Bucket), cursor, i))
	}

	return b.String()
}

// FilteredResources returns resources filtered by bucket and search query.
func FilteredResources(resources []domain.Resource, bucketFilter domain.Bucket, searchQuery string) []domain.Resource {
	filtered := resources

	if bucketFilter != "" {
		var out []domain.Resource
		for _, r := range filtered {
			if r.Bucket == bucketFilter {
				out = append(out, r)
			}
		}
		filtered = out
	}

	if searchQuery != "" {
		q := strings.ToLower(searchQuery)
		var out []domain.Resource
		for _, r := range filtered {
			title := strings.ToLower(r.Title)
			url := strings.ToLower(r.URL)
			if strings.Contains(title, q) || strings.Contains(url, q) {
				out = append(out, r)
			}
		}
		filtered = out
	}

	return filtered
}
