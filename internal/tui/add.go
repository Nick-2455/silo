package tui

import (
	"context"
	"net/url"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Nick-2455/marrow/internal/app"
	"github.com/Nick-2455/marrow/internal/domain"
)

type addState int

const (
	addStateURL addState = iota
	addStateTitle
	addStateSubmitting
	addStateDone
)

// addModel handles the add resource form.
type addModel struct {
	state   addState
	url     string
	title   string
	focused bool
	deps    *app.Deps
}

func newAddModel(deps *app.Deps) *addModel {
	return &addModel{
		state:   addStateURL,
		focused: true,
		deps:    deps,
	}
}

// submitMsg is sent when the form is submitted.
type submitMsg struct {
	id  string
	err error
}

func (m *addModel) Update(msg tea.Msg) (*addModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.state == addStateSubmitting {
			return m, nil
		}

		switch msg.Type {
		case tea.KeyEnter:
			if m.state == addStateURL {
				// Validate URL
				if _, err := url.ParseRequestURI(m.url); err != nil {
					return m, nil // stay on URL field
				}
				m.state = addStateTitle
				return m, nil
			}
			if m.state == addStateTitle {
				m.state = addStateSubmitting
				return m, m.submitCmd()
			}
			if m.state == addStateDone {
				// Reset form
				m.state = addStateURL
				m.url = ""
				m.title = ""
				return m, nil
			}

		case tea.KeyEscape:
			m.state = addStateURL
			m.url = ""
			m.title = ""
			return m, nil

		case tea.KeyBackspace:
			if m.state == addStateURL && len(m.url) > 0 {
				m.url = m.url[:len(m.url)-1]
			} else if m.state == addStateTitle && len(m.title) > 0 {
				m.title = m.title[:len(m.title)-1]
			}

		default:
			if msg.Type == tea.KeyRunes {
				if m.state == addStateURL {
					m.url += string(msg.Runes)
				} else if m.state == addStateTitle {
					m.title += string(msg.Runes)
				}
			}
		}

	case submitMsg:
		if msg.err != nil {
			m.state = addStateURL
			return m, nil
		}
		m.state = addStateDone
		return m, nil
	}

	return m, nil
}

func (m *addModel) View() string {
	var b strings.Builder

	b.WriteString(FormStyle.Render("Add Resource"))
	b.WriteString("\n\n")

	// URL field
	urlLabel := FormLabelStyle.Render("URL:")
	var urlInput string
	if m.state == addStateURL {
		urlInput = FormInputFocusedStyle.Render(m.url + "▌")
	} else {
		urlInput = FormInputStyle.Render(m.url)
	}
	b.WriteString(urlLabel + "\n")
	b.WriteString(urlInput + "\n")

	// Title field
	if m.state != addStateURL {
		b.WriteString("\n")
		titleLabel := FormLabelStyle.Render("Title:")
		var titleInput string
		if m.state == addStateTitle {
			titleInput = FormInputFocusedStyle.Render(m.title + "▌")
		} else {
			titleInput = FormInputStyle.Render(m.title)
		}
		b.WriteString(titleLabel + "\n")
		b.WriteString(titleInput + "\n")
	}

	b.WriteString("\n")

	switch m.state {
	case addStateURL:
		b.WriteString(HelpStyle.Render("Enter URL, then press Enter"))
	case addStateTitle:
		b.WriteString(HelpStyle.Render("Enter title (optional), Enter to submit, Esc to cancel"))
	case addStateSubmitting:
		b.WriteString(HelpStyle.Render("Submitting..."))
	case addStateDone:
		b.WriteString(StatusSuccessStyle.Render("Resource added successfully! Press Enter to add another."))
	}

	return b.String()
}

func (m *addModel) submitCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		resource := domain.Resource{
			URL:   m.url,
			Title: m.title,
			Bucket: domain.BucketInbox,
		}

		id, err := m.deps.Engram.CreateResource(ctx, resource)
		if err != nil {
			return submitMsg{err: err}
		}

		// Set local triage position
		pos := domain.TriagePosition{
			ResourceID: id,
			Bucket:     domain.BucketInbox,
		}
		if err := m.deps.Store.SetTriagePosition(ctx, pos); err != nil {
			return submitMsg{err: err}
		}

		// Invalidate search cache so new resource appears immediately
		_ = m.deps.Store.InvalidateSearchCache(ctx)

		return submitMsg{id: id}
	}
}
