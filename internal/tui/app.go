package tui

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Nick-2455/marrow/internal/app"
	"github.com/Nick-2455/marrow/internal/domain"
)

// ScreenType identifies which TUI screen is active.
type ScreenType int

const (
	ScreenDashboard ScreenType = iota
	ScreenList
	ScreenAdd
	ScreenTriage
	ScreenConfig
)

// StatusType for the status bar.
type StatusType int

const (
	StatusInfo StatusType = iota
	StatusSuccess
	StatusError
	StatusWarning
)

// Screen names for display.
var screenNames = map[ScreenType]string{
	ScreenDashboard: "Dashboard",
	ScreenList:      "List",
	ScreenAdd:       "Add",
	ScreenTriage:    "Triage",
	ScreenConfig:    "Config",
}

// Navigation keys.
type navKeys struct {
	Tab       key.Binding
	ShiftTab  key.Binding
	Up        key.Binding
	Down      key.Binding
	Enter     key.Binding
	Escape    key.Binding
	Quit      key.Binding
	Help      key.Binding
	Search    key.Binding
	New       key.Binding
	Triage    key.Binding
	Dashboard key.Binding
	Filter    key.Binding
}

func (k navKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.Tab, k.Quit, k.Help}
}

func (k navKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Tab, k.ShiftTab, k.Enter, k.Escape},
		{k.Up, k.Down, k.Quit, k.Help},
		{k.Search, k.New, k.Triage, k.Dashboard},
	}
}

func defaultNavKeys() navKeys {
	return navKeys{
		Tab:       key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next screen")),
		ShiftTab:  key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "prev screen")),
		Up:        key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:        key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Enter:     key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
		Escape:    key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		Quit:      key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		Help:      key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "toggle help")),
		Search:    key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
		New:       key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new resource")),
		Triage:    key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "triage")),
		Dashboard: key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "dashboard")),
		Filter:    key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "filter bucket")),
	}
}

// tickMsg triggers periodic health checks.
type tickMsg struct{}

// healthResultMsg carries the result of a health check.
type HealthResultMsg struct {
	OK bool
}

// errorMsg carries an error to display in the status bar.
type ErrorMsg string

// successMsg carries a success message.
type SuccessMsg string

// Model is the root Bubble Tea model.
type Model struct {
	screen      ScreenType
	resources   []domain.Resource
	filtered    []domain.Resource
	searchQuery string
	bucketFilter domain.Bucket
	config      domain.Config
	engramOK    bool
	status      string
	statusType  StatusType
	help        help.Model
	keys        navKeys
	showHelp    bool

	// Sub-models
	listModel    *listModel
	addModel     *addModel
	triageModel  *triageModel
	dashModel    *dashboardModel
	configModel  *configScreenModel

	// Dependencies
	store  *app.Deps
	loader domain.ConfigLoader

	// Health check timer
	healthInterval time.Duration
}

// NewModel creates a new root TUI model.
func NewModel(deps *app.Deps) *Model {
	m := &Model{
		screen:         ScreenDashboard,
		engramOK:       true,
		help:           help.New(),
		keys:           defaultNavKeys(),
		store:          deps,
		loader:         deps.Loader,
		healthInterval: 30 * time.Second,
	}

	// Load config
	cfg, err := deps.Loader.Load()
	if err == nil {
		m.config = cfg
	}

	// Initialize sub-models
	m.listModel = newListModel(deps)
	m.addModel = newAddModel(deps)
	m.triageModel = newTriageModel(deps)
	m.dashModel = newDashboardModel(deps)
	m.configModel = newConfigScreenModel(deps)

	return m
}

// Init implements tea.Model.
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.tickCmd(),
		m.healthCheckCmd(),
		m.refreshResourcesCmd(),
	)
}

// Update implements tea.Model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)

	case tickMsg:
		cmds = append(cmds, m.tickCmd(), m.healthCheckCmd())

	case HealthResultMsg:
		m.engramOK = msg.OK
		if !msg.OK {
			m.status = "Engram unreachable — using cached data"
			m.statusType = StatusWarning
		}

	case ErrorMsg:
		m.status = string(msg)
		m.statusType = StatusError

	case SuccessMsg:
		m.status = string(msg)
		m.statusType = StatusSuccess

	case ResourcesLoadedMsg:
		m.resources = msg.Resources
		m.applyFilters()
	}

	// Update active sub-model
	var cmd tea.Cmd
	switch m.screen {
	case ScreenList:
		m.listModel, cmd = m.listModel.Update(msg)
	case ScreenAdd:
		m.addModel, cmd = m.addModel.Update(msg)
	case ScreenTriage:
		m.triageModel, cmd = m.triageModel.Update(msg)
	case ScreenDashboard:
		m.dashModel, cmd = m.dashModel.Update(msg)
	case ScreenConfig:
		m.configModel, cmd = m.configModel.Update(msg)
	}
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View implements tea.Model.
func (m *Model) View() string {
	var content string

	switch m.screen {
	case ScreenList:
		content = m.listModel.View()
	case ScreenAdd:
		content = m.addModel.View()
	case ScreenTriage:
		content = m.triageModel.View()
	case ScreenDashboard:
		content = m.dashModel.View()
	case ScreenConfig:
		content = m.configModel.View()
	}

	header := m.renderHeader()
	footer := m.renderFooter()

	return AppStyle.Render(lipgloss.JoinVertical(lipgloss.Left, header, content, footer))
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global quit
	if key.Matches(msg, m.keys.Quit) {
		return m, tea.Quit
	}

	// Toggle help
	if key.Matches(msg, m.keys.Help) {
		m.showHelp = !m.showHelp
		return m, nil
	}

	// Screen navigation (only when not in a focused input)
	if !m.isInputFocused() {
		if key.Matches(msg, m.keys.Tab) {
			m.nextScreen()
			return m, m.refreshCurrentScreen()
		}
		if key.Matches(msg, m.keys.ShiftTab) {
			m.prevScreen()
			return m, m.refreshCurrentScreen()
		}
		// Direct screen shortcuts
		switch msg.String() {
		case "d":
			if m.screen != ScreenDashboard {
				m.screen = ScreenDashboard
				return m, m.refreshCurrentScreen()
			}
		case "l":
			if m.screen != ScreenList {
				m.screen = ScreenList
				return m, m.refreshCurrentScreen()
			}
		case "n":
			if m.screen != ScreenAdd {
				m.screen = ScreenAdd
				return m, nil
			}
		case "t":
			if m.screen != ScreenTriage {
				m.screen = ScreenTriage
				return m, m.refreshCurrentScreen()
			}
		}
	}

	// Escape goes back to dashboard
	if key.Matches(msg, m.keys.Escape) && m.screen != ScreenDashboard && !m.isInputFocused() {
		m.screen = ScreenDashboard
		return m, m.refreshCurrentScreen()
	}

	return m, nil
}

func (m *Model) isInputFocused() bool {
	switch m.screen {
	case ScreenAdd:
		return m.addModel != nil && m.addModel.focused
	case ScreenConfig:
		return m.configModel != nil && m.configModel.focused
	case ScreenList:
		return m.listModel != nil && m.listModel.searchFocused
	}
	return false
}

func (m *Model) nextScreen() {
	m.screen = (m.screen + 1) % 5
}

func (m *Model) prevScreen() {
	m.screen = (m.screen + 4) % 5 // -1 mod 5
}

// HealthInterval returns the health check interval.
func (m *Model) HealthInterval() time.Duration {
	return m.healthInterval
}

// StatusType returns the current status bar type.
func (m *Model) StatusType() StatusType {
	return m.statusType
}

// TickCmd returns a command that triggers a periodic tick.
func (m *Model) TickCmd() tea.Cmd {
	return m.tickCmd()
}

func (m *Model) tickCmd() tea.Cmd {
	return tea.Tick(m.healthInterval, func(_ time.Time) tea.Msg {
		return tickMsg{}
	})
}

func (m *Model) healthCheckCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		ok := m.store.Engram.IsReachable(ctx)
		return HealthResultMsg{OK: ok}
	}
}

func (m *Model) refreshResourcesCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Load all triage positions and build resource list
		positions, err := m.store.Store.GetAllTriagePositions(ctx)
		if err != nil {
			return ErrorMsg(fmt.Sprintf("failed to load resources: %v", err))
		}

		resources := make([]domain.Resource, 0, len(positions))
		for _, pos := range positions {
			r := domain.Resource{
				ID:     pos.ResourceID,
				Bucket: pos.Bucket,
			}
			// Try to get full resource from Engram
			full, err := m.store.Engram.GetResource(ctx, pos.ResourceID)
			if err == nil {
				r = full
			}
			resources = append(resources, r)
		}

		return ResourcesLoadedMsg{Resources: resources}
	}
}

func (m *Model) refreshCurrentScreen() tea.Cmd {
	switch m.screen {
	case ScreenList:
		m.listModel.resources = m.filtered
		return nil
	case ScreenTriage:
		m.triageModel.resources = m.filtered
		return nil
	case ScreenDashboard:
		m.dashModel.resources = m.resources
		return nil
	}
	return nil
}

func (m *Model) applyFilters() {
	filtered := m.resources

	// Apply bucket filter
	if m.bucketFilter != "" {
		var bucketFiltered []domain.Resource
		for _, r := range filtered {
			if r.Bucket == m.bucketFilter {
				bucketFiltered = append(bucketFiltered, r)
			}
		}
		filtered = bucketFiltered
	}

	// Apply text search
	if m.searchQuery != "" {
		q := m.searchQuery
		var textFiltered []domain.Resource
		for _, r := range filtered {
			if containsIgnoreCase(r.Title, q) || containsIgnoreCase(r.URL, q) {
				textFiltered = append(textFiltered, r)
			}
		}
		filtered = textFiltered
	}

	m.filtered = filtered
}

func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) && searchLower(s, substr)
}

func searchLower(s, substr string) bool {
	s = toLower(s)
	substr = toLower(substr)
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		result[i] = c
	}
	return string(result)
}

func (m *Model) renderHeader() string {
	title := "marrow"
	screen := screenNames[m.screen]

	parts := []string{title, screen}
	if !m.engramOK {
		parts = append(parts, DegradedIndicator())
	}

	return HeaderStyle.Render(lipgloss.JoinHorizontal(lipgloss.Left, parts...))
}

func (m *Model) renderFooter() string {
	var statusStyle lipgloss.Style
	switch m.statusType {
	case StatusSuccess:
		statusStyle = StatusSuccessStyle
	case StatusError:
		statusStyle = StatusErrorStyle
	case StatusWarning:
		statusStyle = StatusWarnStyle
	default:
		statusStyle = StatusInfoStyle
	}

	status := statusStyle.Render(m.status)

	// Tabs
	var tabs []string
	for _, s := range []ScreenType{ScreenDashboard, ScreenList, ScreenAdd, ScreenTriage, ScreenConfig} {
		name := screenNames[s]
		if s == m.screen {
			tabs = append(tabs, TabActiveStyle.Render(name))
		} else {
			tabs = append(tabs, TabStyle.Render(name))
		}
	}

	row := lipgloss.JoinHorizontal(lipgloss.Center, status, "  ", lipgloss.JoinHorizontal(lipgloss.Left, tabs...))
	return StatusBarStyle.Render(row)
}

// ResourcesLoadedMsg is sent when resources are loaded from the store.
type ResourcesLoadedMsg struct {
	Resources []domain.Resource
}
