package tui

import (
	"context"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Nick-2455/marrow/internal/app"
	"github.com/Nick-2455/marrow/internal/domain"
)

type triageState int

const (
	triageStateSelect triageState = iota
	triageStateChooseBucket
	triageStateMoving
	triageStateDone
)

// triageModel handles the triage screen.
type triageModel struct {
	resources []domain.Resource
	cursor    int
	state     triageState
	bucketCursor int
	deps      *app.Deps
}

func newTriageModel(deps *app.Deps) *triageModel {
	return &triageModel{
		state: triageStateSelect,
		deps:  deps,
	}
}

// triageResultMsg is sent when triage operation completes.
type triageResultMsg struct {
	err error
}

func (m *triageModel) Update(msg tea.Msg) (*triageModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case triageStateSelect:
			switch msg.String() {
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down", "j":
				if m.cursor < len(m.resources)-1 {
					m.cursor++
				}
			case "enter":
				if len(m.resources) > 0 {
					m.state = triageStateChooseBucket
					m.bucketCursor = 0
				}
			}

		case triageStateChooseBucket:
			buckets := domain.AllBuckets()
			switch msg.String() {
			case "up", "k":
				if m.bucketCursor > 0 {
					m.bucketCursor--
				}
			case "down", "j":
				if m.bucketCursor < len(buckets)-1 {
					m.bucketCursor++
				}
			case "enter":
				m.state = triageStateMoving
				return m, m.moveCmd(buckets[m.bucketCursor])
			case "esc":
				m.state = triageStateSelect
			}

		case triageStateDone:
			if msg.Type == tea.KeyEnter || msg.String() == "esc" {
				m.state = triageStateSelect
			}
		}

	case triageResultMsg:
		m.state = triageStateDone
		if msg.err != nil {
			return m, nil
		}
	}

	return m, nil
}

func (m *triageModel) View() string {
	var b strings.Builder

	b.WriteString(FormStyle.Render(
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(ColorPrimary)).Render("Triage"),
	))
	b.WriteString("\n\n")

	switch m.state {
	case triageStateSelect:
		if len(m.resources) == 0 {
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(ColorMuted)).Render("No resources to triage"))
			b.WriteString("\n\n")
			b.WriteString(HelpStyle.Render("Add resources first, then return here"))
			return b.String()
		}

		b.WriteString(FormLabelStyle.Render("Select a resource:"))
		b.WriteString("\n\n")

		for i, r := range m.resources {
			var item string
			title := ListTitleStyle.Render(truncate(r.Title, 40))
			bucket := ListBucketStyle.Render("[" + string(r.Bucket) + "]")
			item = lipgloss.JoinHorizontal(lipgloss.Left, bucket, " ", title)

			if i == m.cursor {
				b.WriteString(TriageSelectedStyle.Render(item))
			} else {
				b.WriteString(TriageOptionStyle.Render(item))
			}
			b.WriteString("\n")
		}

		b.WriteString("\n")
		b.WriteString(HelpStyle.Render("↑/↓ select  Enter to choose target bucket"))

	case triageStateChooseBucket:
		b.WriteString(FormLabelStyle.Render("Move to bucket:"))
		b.WriteString("\n\n")

		buckets := domain.AllBuckets()
		for i, bucket := range buckets {
			var item string
			if i == m.bucketCursor {
				item = TriageSelectedStyle.Render(string(bucket))
			} else {
				item = TriageOptionStyle.Render(string(bucket))
			}
			b.WriteString(item)
			b.WriteString("\n")
		}

		b.WriteString("\n")
		b.WriteString(HelpStyle.Render("↑/↓ select  Enter to move  Esc to cancel"))

	case triageStateMoving:
		b.WriteString(HelpStyle.Render("Moving resource..."))

	case triageStateDone:
		b.WriteString(StatusSuccessStyle.Render("Resource moved successfully!"))
		b.WriteString("\n\n")
		b.WriteString(HelpStyle.Render("Press Enter or Esc to continue"))
	}

	return b.String()
}

func (m *triageModel) moveCmd(targetBucket domain.Bucket) tea.Cmd {
	if m.cursor >= len(m.resources) {
		return func() tea.Msg {
			return triageResultMsg{err: nil}
		}
	}

	resource := m.resources[m.cursor]

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Get current position for rollback
		oldPos, err := m.deps.Store.GetTriagePosition(ctx, resource.ID)
		if err != nil && err != domain.ErrTriageNotFound {
			return triageResultMsg{err: err}
		}

		// Update SQLite first
		newPos := domain.TriagePosition{
			ResourceID: resource.ID,
			Bucket:     targetBucket,
		}
		if err := m.deps.Store.SetTriagePosition(ctx, newPos); err != nil {
			return triageResultMsg{err: err}
		}

		// Update Engram
		if err := m.deps.Engram.UpdateResource(ctx, resource.ID, map[string]any{
			"bucket": string(targetBucket),
		}); err != nil {
			// Rollback SQLite
			if oldPos.ResourceID != "" {
				_ = m.deps.Store.SetTriagePosition(ctx, oldPos)
			}
			return triageResultMsg{err: err}
		}

		return triageResultMsg{}
	}
}
