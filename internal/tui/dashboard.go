package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Nick-2455/marrow/internal/app"
	"github.com/Nick-2455/marrow/internal/domain"
)

const recentItemsPerBucket = 5

// dashboardModel handles the roadmap dashboard.
type dashboardModel struct {
	resources []domain.Resource
	deps      *app.Deps
}

func newDashboardModel(deps *app.Deps) *dashboardModel {
	return &dashboardModel{
		deps: deps,
	}
}

func (m *dashboardModel) Update(msg tea.Msg) (*dashboardModel, tea.Cmd) {
	return m, nil
}

func (m *dashboardModel) View() string {
	var b strings.Builder

	b.WriteString(FormStyle.Render("Roadmap Dashboard"))
	b.WriteString("\n\n")

	// Group resources by bucket
	buckets := make(map[domain.Bucket][]domain.Resource)
	for _, r := range m.resources {
		buckets[r.Bucket] = append(buckets[r.Bucket], r)
	}

	// Show each bucket
	for _, bucket := range domain.AllBuckets() {
		items := buckets[bucket]
		count := len(items)

		// Bucket header with count
		header := lipgloss.JoinHorizontal(lipgloss.Left,
			DashBucketStyle.Render(string(bucket)),
			" ",
			DashCountStyle.Render(fmt.Sprintf("(%d)", count)),
		)
		b.WriteString(header)
		b.WriteString("\n")

		// Recent items
		if count == 0 {
			b.WriteString(DashItemStyle.Render("  (empty)"))
		} else {
			limit := recentItemsPerBucket
			if count < limit {
				limit = count
			}
			for i := 0; i < limit; i++ {
				r := items[i]
				title := truncate(r.Title, 45)
				if r.Title == "" {
					title = truncate(r.URL, 45)
				}
				b.WriteString(DashItemStyle.Render("  • " + title))
			}
			if count > recentItemsPerBucket {
				b.WriteString(DashItemStyle.Render(
					fmt.Sprintf("  ... and %d more", count-recentItemsPerBucket),
				))
			}
		}

		b.WriteString("\n\n")
	}

	total := len(m.resources)
	b.WriteString(HelpStyle.Render(
		fmt.Sprintf("Total: %d resources  Press 'l' for list, 't' for triage, 'n' to add", total),
	))

	return b.String()
}
