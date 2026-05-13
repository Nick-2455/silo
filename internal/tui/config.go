package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Nick-2455/marrow/internal/app"
	"github.com/Nick-2455/marrow/internal/domain"
)

type configField int

const (
	configFieldProfile configField = iota
	configFieldEngramAPI
	configFieldEngramKey
	configFieldTriageModel
	configFieldSummaryModel
)

var configFieldNames = map[configField]string{
	configFieldProfile:     "Profile",
	configFieldEngramAPI:   "Engram API URL",
	configFieldEngramKey:   "Engram API Key",
	configFieldTriageModel: "Triage Model",
	configFieldSummaryModel: "Summary Model",
}

// configScreenModel handles the config edit screen.
type configScreenModel struct {
	config     domain.Config
	cursor     configField
	focused    bool
	editing    bool
	editBuffer string
	deps       *app.Deps
	loader     domain.ConfigLoader
	saved      bool
}

func newConfigScreenModel(deps *app.Deps) *configScreenModel {
	cfg, _ := deps.Loader.Load()
	return &configScreenModel{
		config: cfg,
		deps:   deps,
		loader: deps.Loader,
	}
}

func (m *configScreenModel) Update(msg tea.Msg) (*configScreenModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.editing {
			switch msg.Type {
			case tea.KeyEnter:
				m.applyEdit()
				m.editing = false
				m.editBuffer = ""
				m.focused = false
				return m, nil
			case tea.KeyEscape:
				m.editing = false
				m.editBuffer = ""
				m.focused = false
				return m, nil
			case tea.KeyBackspace:
				if len(m.editBuffer) > 0 {
					m.editBuffer = m.editBuffer[:len(m.editBuffer)-1]
				}
			default:
				if msg.Type == tea.KeyRunes {
					m.editBuffer += string(msg.Runes)
				}
			}
			return m, nil
		}

		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < configFieldSummaryModel {
				m.cursor++
			}
		case "enter":
			m.focused = true
			m.editing = true
			m.editBuffer = m.getFieldValue(m.cursor)
		case "s":
			m.saveConfig()
		}
	}

	return m, nil
}

func (m *configScreenModel) View() string {
	var b strings.Builder

	b.WriteString(FormStyle.Render(
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(ColorPrimary)).Render("Configuration"),
	))
	b.WriteString("\n\n")

	fields := []configField{
		configFieldProfile,
		configFieldEngramAPI,
		configFieldEngramKey,
		configFieldTriageModel,
		configFieldSummaryModel,
	}

	for _, f := range fields {
		key := ConfigKeyStyle.Render(configFieldNames[f] + ":")
		var value string

		if m.cursor == f && m.editing {
			value = FormInputFocusedStyle.Render(m.editBuffer + "▌")
		} else {
			val := m.getFieldValue(f)
			if f == configFieldEngramKey && val != "" {
				val = maskString(val)
			}
			value = ConfigValueStyle.Render(val)
		}

		row := lipgloss.JoinHorizontal(lipgloss.Left, key, " ", value)
		if m.cursor == f && !m.editing {
			row = lipgloss.NewStyle().
				Background(lipgloss.Color(ColorSurface)).
				Border(lipgloss.NormalBorder(), false, false, false, true).
				BorderForeground(lipgloss.Color(ColorPrimary)).
				Padding(0, 1).
				Render(row)
		}

		b.WriteString(row)
		b.WriteString("\n")
	}

	b.WriteString("\n")

	if m.saved {
		b.WriteString(StatusSuccessStyle.Render("Configuration saved!"))
		b.WriteString("\n\n")
	}

	b.WriteString(HelpStyle.Render("↑/↓ navigate  Enter to edit  s to save  Esc to cancel"))

	return b.String()
}

func (m *configScreenModel) getFieldValue(f configField) string {
	switch f {
	case configFieldProfile:
		return m.config.Profile
	case configFieldEngramAPI:
		return m.config.EngramAPI
	case configFieldEngramKey:
		return m.config.EngramKey
	case configFieldTriageModel:
		return m.config.ModelPrefs.Triage
	case configFieldSummaryModel:
		return m.config.ModelPrefs.Summary
	}
	return ""
}

func (m *configScreenModel) applyEdit() {
	switch m.cursor {
	case configFieldProfile:
		m.config.Profile = m.editBuffer
	case configFieldEngramAPI:
		m.config.EngramAPI = m.editBuffer
	case configFieldEngramKey:
		m.config.EngramKey = m.editBuffer
	case configFieldTriageModel:
		m.config.ModelPrefs.Triage = m.editBuffer
	case configFieldSummaryModel:
		m.config.ModelPrefs.Summary = m.editBuffer
	}
}

func (m *configScreenModel) saveConfig() {
	if err := m.loader.Save(m.config); err != nil {
		// Error would be shown in status bar via parent model
		return
	}
	m.saved = true
}

func maskString(s string) string {
	if len(s) <= 8 {
		return "****"
	}
	return s[:4] + "****" + s[len(s)-4:]
}
