package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Nick-2455/marrow/internal/app"
	"github.com/Nick-2455/marrow/internal/domain"
)

// listModel handles the resource list screen.
type listModel struct {
	resources     []domain.Resource
	cursor        int
	searchFocused bool
	searchInput   textinput.Model
	bucketFilter  domain.Bucket
	deps          *app.Deps
}

func newListModel(deps *app.Deps) *listModel {
	ti := textinput.New()
	ti.Placeholder = "Search resources..."
	ti.CharLimit = 128
	ti.Width = 50
	ti.Prompt = "/ "

	return &listModel{
		searchInput: ti,
		deps:        deps,
	}
}

func (m *listModel) Update(msg tea.Msg) (*listModel, tea.Cmd) {
	var cmds []tea.Cmd

	if m.searchFocused {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.Type {
			case tea.KeyEnter, tea.KeyEscape:
				m.searchFocused = false
				return m, nil
			}
		}
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "/":
			m.searchFocused = true
			m.searchInput.SetValue("")
			var cmd tea.Cmd
			m.searchInput, cmd = m.searchInput.Update(msg)
			cmds = append(cmds, cmd)
			return m, tea.Batch(cmds...)
		case "f":
			m.cycleBucketFilter()
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.resources)-1 {
				m.cursor++
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *listModel) View() string {
	var b strings.Builder

	// Search bar
	if m.searchFocused {
		b.WriteString(FormInputFocusedStyle.Render(m.searchInput.View()))
	} else {
		b.WriteString(FormInputStyle.Render(m.searchInput.View()))
	}
	b.WriteString("\n\n")

	// Bucket filter indicator
	if m.bucketFilter != "" {
		filterLabel := ListBucketStyle.Render("Filter: " + string(m.bucketFilter))
		b.WriteString(filterLabel)
		b.WriteString("\n\n")
	}

	// Resource list
	if len(m.resources) == 0 {
		b.WriteString("No resources found")
		b.WriteString("\n\n")
		b.WriteString(HelpStyle.Render("Press 'f' to filter by bucket, '/' to search, 'n' to add"))
		return b.String()
	}

	for i, r := range m.resources {
		var item string
		title := ListTitleStyle.Render(truncate(r.Title, 50))
		url := ListURLStyle.Render(truncate(r.URL, 60))
		bucket := ListBucketStyle.Render("[" + string(r.Bucket) + "]")

		item = lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.JoinHorizontal(lipgloss.Left, bucket, " ", title),
			url,
		)

		if i == m.cursor {
			b.WriteString(ListItemSelectedStyle.Render(item))
		} else {
			b.WriteString(ListItemStyle.Render(item))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(HelpStyle.Render("↑/↓ navigate  f filter  / search  n add  t triage"))

	return b.String()
}

func (m *listModel) cycleBucketFilter() {
	buckets := domain.AllBuckets()
	idx := -1
	for i, b := range buckets {
		if b == m.bucketFilter {
			idx = i
			break
		}
	}
	if idx == -1 {
		m.bucketFilter = buckets[0]
	} else if idx == len(buckets)-1 {
		m.bucketFilter = "" // clear filter
	} else {
		m.bucketFilter = buckets[idx+1]
	}
	m.cursor = 0
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
