package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Nick-2455/marrow/internal/app"
	"github.com/Nick-2455/marrow/internal/domain"
	"github.com/Nick-2455/marrow/internal/tui/screens"
	"github.com/Nick-2455/marrow/internal/tui/styles"
)

const (
	sepLine        = "────────────────────────────────────────────────────"
	maxTitleLen    = 50
	maxURLLen      = 55
	healthInterval = 30 * time.Second
)

// Model is the root Bubble Tea model.
type Model struct {
	Screen   Screen
	Previous Screen
	Cursor   int
	Width    int
	Height   int

	// Data
	Resources []domain.Resource
	Config    domain.Config
	EngramOK  bool
	StatusMsg string

	// Add screen state
	AddURL   string
	AddTitle string
	AddStep  int // 0=url, 1=title, 2=submitting

	// List screen state
	SearchQuery  string
	Searching    bool
	BucketFilter domain.Bucket

	// Triage screen state
	TriageResource domain.Resource
	BucketCursor   int
	TriageMoving   bool
	TriageDone     bool
	TriageErr      string

	// Dependencies
	Deps *app.Deps
}

// NewModel creates a new TUI model.
func NewModel(deps *app.Deps) tea.Model {
	cfg, _ := deps.Loader.Load()
	return &Model{
		Screen:   ScreenDashboard,
		Previous: ScreenDashboard,
		EngramOK: true,
		Config:   cfg,
		Deps:     deps,
	}
}

// Init implements tea.Model.
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.refreshResourcesCmd(),
		m.healthCheckCmd(),
		m.tickCmd(),
	)
}

// Update implements tea.Model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)

	case tickMsg:
		return m, tea.Batch(m.tickCmd(), m.healthCheckCmd())

	case HealthResultMsg:
		m.EngramOK = msg.OK
		if !msg.OK {
			m.StatusMsg = "Engram unreachable — using cached data"
		}

	case ResourceLoadedMsg:
		m.Resources = msg.Resources
		if msg.Err != "" {
			m.StatusMsg = msg.Err
		}

	case AddSubmitMsg:
		m.AddStep = 0
		if msg.Err != "" {
			m.StatusMsg = "Error: " + msg.Err
		} else {
			m.StatusMsg = "Resource added"
			m.AddURL = ""
			m.AddTitle = ""
			return m, m.refreshResourcesCmd()
		}

	case TriageMoveMsg:
		m.TriageMoving = false
		if msg.Err != "" {
			m.TriageErr = msg.Err
			m.TriageDone = false
			m.StatusMsg = "Error: " + msg.Err
		} else {
			m.TriageDone = true
			m.StatusMsg = "Resource moved to " + string(msg.Bucket)
			return m, m.refreshResourcesCmd()
		}
	}

	return m, nil
}

// View implements tea.Model.
func (m *Model) View() string {
	var b strings.Builder

	b.WriteString(m.header())
	b.WriteString("\n")
	b.WriteString(sepLine)
	b.WriteString("\n\n")

	switch m.Screen {
	case ScreenDashboard:
		b.WriteString(screens.RenderDashboard(m.Resources, m.Cursor))
	case ScreenList:
		b.WriteString(screens.RenderList(m.Resources, m.BucketFilter, m.SearchQuery, m.Searching, m.Cursor))
	case ScreenAdd:
		b.WriteString(screens.RenderAdd(m.AddURL, m.AddTitle, m.AddStep))
	case ScreenTriage:
		b.WriteString(screens.RenderTriage(m.TriageResource, m.BucketCursor, m.TriageMoving, m.TriageDone, m.TriageErr))
	case ScreenConfig:
		b.WriteString(screens.RenderConfig(m.Config, m.Cursor))
	}

	b.WriteString("\n")
	b.WriteString(sepLine)
	b.WriteString("\n")
	b.WriteString(m.footer())

	return styles.FrameStyle.Render(b.String())
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global quit
	if msg.String() == "q" || msg.String() == "ctrl+c" || msg.Type == tea.KeyCtrlC {
		return m, tea.Quit
	}

	// Global shortcuts (not when typing in add form)
	if !m.isInputFocused() {
		switch msg.String() {
		case "d":
			if m.Screen != ScreenDashboard {
				m.setScreen(ScreenDashboard)
				return m, m.refreshResourcesCmd()
			}
		case "a":
			if m.Screen != ScreenAdd {
				m.Previous = m.Screen
				m.Screen = ScreenAdd
				m.Cursor = 0
				m.AddStep = 0
				m.AddURL = ""
				m.AddTitle = ""
				return m, nil
			}
		case "t":
			if m.Screen != ScreenTriage && len(m.Resources) > 0 {
				m.Previous = m.Screen
				m.Screen = ScreenTriage
				m.Cursor = 0
				m.TriageDone = false
				m.TriageErr = ""
				m.TriageMoving = false
				return m, nil
			}
		case "c":
			if m.Screen != ScreenConfig {
				m.setScreen(ScreenConfig)
				return m, nil
			}
		}
	}

	// ESC: go back
	if msg.Type == tea.KeyEscape {
		return m.handleEscape()
	}

	// Screen-specific handling
	switch m.Screen {
	case ScreenDashboard:
		return m, m.handleDashboardKey(msg)
	case ScreenList:
		return m, m.handleListKey(msg)
	case ScreenAdd:
		return m, m.handleAddKey(msg)
	case ScreenTriage:
		return m, m.handleTriageKey(msg)
	case ScreenConfig:
		return m, m.handleConfigKey(msg)
	}

	return m, nil
}

func (m *Model) handleEscape() (tea.Model, tea.Cmd) {
	switch m.Screen {
	case ScreenList:
		m.Screen = ScreenDashboard
		m.Cursor = 0
		m.BucketFilter = ""
		m.SearchQuery = ""
		m.Searching = false
		return m, nil
	case ScreenAdd:
		m.Screen = ScreenDashboard
		m.AddStep = 0
		m.AddURL = ""
		m.AddTitle = ""
		return m, nil
	case ScreenTriage:
		if m.TriageMoving {
			return m, nil // don't escape while moving
		}
		m.Screen = m.Previous
		if m.Screen == ScreenTriage || m.Screen == ScreenAdd {
			m.Screen = ScreenDashboard
		}
		m.Cursor = 0
		m.TriageDone = false
		m.TriageErr = ""
		return m, nil
	case ScreenConfig:
		m.Screen = ScreenDashboard
		m.Cursor = 0
		return m, nil
	default:
		return m, nil
	}
}

func (m *Model) isInputFocused() bool {
	return m.Screen == ScreenAdd
}

// ── Dashboard ──────────────────────────────────────────────────────────────

func (m *Model) handleDashboardKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "up", "k":
		if m.Cursor > 0 {
			m.Cursor--
		}
	case "down", "j":
		if m.Cursor < 3 {
			m.Cursor++
		}
	case "enter":
		buckets := domain.AllBuckets()
		if m.Cursor < len(buckets) {
			m.BucketFilter = buckets[m.Cursor]
			m.Previous = ScreenDashboard
			m.Screen = ScreenList
			m.Cursor = 0
		}
	}
	return nil
}

// ── List ───────────────────────────────────────────────────────────────────

func (m *Model) handleListKey(msg tea.KeyMsg) tea.Cmd {
	if m.Searching {
		return m.handleListSearch(msg)
	}

	switch msg.String() {
	case "/":
		m.Searching = true
		m.SearchQuery = ""
		return nil
	case "up", "k":
		if m.Cursor > 0 {
			m.Cursor--
		}
	case "down", "j":
		filtered := m.filteredResources()
		if m.Cursor < len(filtered)-1 {
			m.Cursor++
		}
	case "enter":
		filtered := m.filteredResources()
		if len(filtered) > 0 && m.Cursor < len(filtered) {
			m.TriageResource = filtered[m.Cursor]
			m.Previous = ScreenList
			m.Screen = ScreenTriage
			m.BucketCursor = 0
			m.TriageDone = false
			m.TriageErr = ""
			m.TriageMoving = false
		}
	}
	return nil
}

func (m *Model) handleListSearch(msg tea.KeyMsg) tea.Cmd {
	switch msg.Type {
	case tea.KeyEnter, tea.KeyEscape:
		m.Searching = false
		m.Cursor = 0
	case tea.KeyBackspace:
		if len(m.SearchQuery) > 0 {
			m.SearchQuery = m.SearchQuery[:len(m.SearchQuery)-1]
		}
	default:
		if msg.Type == tea.KeyRunes {
			m.SearchQuery += string(msg.Runes)
		}
	}
	return nil
}

func (m *Model) filteredResources() []domain.Resource {
	return screens.FilteredResources(m.Resources, m.BucketFilter, m.SearchQuery)
}

// ── Add ────────────────────────────────────────────────────────────────────

func (m *Model) handleAddKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.Type {
	case tea.KeyEnter:
		if m.AddStep == 0 {
			if m.AddURL == "" {
				return nil
			}
			m.AddStep = 1
			return nil
		}
		if m.AddStep == 1 {
			m.AddStep = 2 // submitting
			return m.submitAddCmd()
		}
	case tea.KeyBackspace:
		if m.AddStep == 0 && len(m.AddURL) > 0 {
			m.AddURL = m.AddURL[:len(m.AddURL)-1]
		} else if m.AddStep == 1 && len(m.AddTitle) > 0 {
			m.AddTitle = m.AddTitle[:len(m.AddTitle)-1]
		}
	case tea.KeyRunes:
		if m.AddStep == 0 {
			m.AddURL += string(msg.Runes)
		} else if m.AddStep == 1 {
			m.AddTitle += string(msg.Runes)
		}
	}
	return nil
}

func (m *Model) submitAddCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		resource := domain.Resource{
			URL:    m.AddURL,
			Title:  m.AddTitle,
			Bucket: domain.BucketInbox,
		}

		id, err := m.Deps.Engram.CreateResource(ctx, resource)
		if err != nil {
			return AddSubmitMsg{Err: err.Error()}
		}

		pos := domain.TriagePosition{
			ResourceID: id,
			Bucket:     domain.BucketInbox,
		}
		if err := m.Deps.Store.SetTriagePosition(ctx, pos); err != nil {
			return AddSubmitMsg{Err: err.Error()}
		}

		_ = m.Deps.Store.InvalidateSearchCache(ctx)

		return AddSubmitMsg{ID: id}
	}
}

// ── Triage ─────────────────────────────────────────────────────────────────

func (m *Model) handleTriageKey(msg tea.KeyMsg) tea.Cmd {
	if m.TriageMoving || m.TriageDone {
		return nil
	}

	buckets := domain.AllBuckets()

	switch msg.String() {
	case "up", "k":
		if m.BucketCursor > 0 {
			m.BucketCursor--
		}
	case "down", "j":
		if m.BucketCursor < len(buckets)-1 {
			m.BucketCursor++
		}
	case "enter":
		m.TriageMoving = true
		return m.moveResourceCmd(buckets[m.BucketCursor])
	}

	return nil
}

func (m *Model) moveResourceCmd(target domain.Bucket) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		r := m.TriageResource

		// Save old position for rollback
		oldPos, err := m.Deps.Store.GetTriagePosition(ctx, r.ID)
		if err != nil && err != domain.ErrTriageNotFound {
			return TriageMoveMsg{Err: err.Error()}
		}

		// Update SQLite first
		newPos := domain.TriagePosition{
			ResourceID: r.ID,
			Bucket:     target,
		}
		if err := m.Deps.Store.SetTriagePosition(ctx, newPos); err != nil {
			return TriageMoveMsg{Err: err.Error()}
		}

		// Update Engram — pass ALL fields because mem_update replaces content, not merges
		if err := m.Deps.Engram.UpdateResource(ctx, r.ID, map[string]any{
			"url":    r.URL,
			"title":  r.Title,
			"bucket": string(target),
		}); err != nil {
			// Rollback SQLite
			if oldPos.ResourceID != "" {
				_ = m.Deps.Store.SetTriagePosition(ctx, oldPos)
			}
			return TriageMoveMsg{Err: err.Error()}
		}

		return TriageMoveMsg{Bucket: target}
	}
}

// ── Config ─────────────────────────────────────────────────────────────────

func (m *Model) handleConfigKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "up", "k":
		if m.Cursor > 0 {
			m.Cursor--
		}
	case "down", "j":
		if m.Cursor < 3 {
			m.Cursor++
		}
	}
	return nil
}

// ── Commands ───────────────────────────────────────────────────────────────

func (m *Model) refreshResourcesCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		positions, err := m.Deps.Store.GetAllTriagePositions(ctx)
		if err != nil {
			return ResourceLoadedMsg{Err: fmt.Sprintf("load resources: %v", err)}
		}

		resources := make([]domain.Resource, 0, len(positions))
		for _, pos := range positions {
			r := domain.Resource{
				ID:     pos.ResourceID,
				Bucket: pos.Bucket,
			}
			full, err := m.Deps.Engram.GetResource(ctx, pos.ResourceID)
			if err == nil {
				r = full
			}
			resources = append(resources, r)
		}

		return ResourceLoadedMsg{Resources: resources}
	}
}

func (m *Model) healthCheckCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		ok := m.Deps.Engram.IsReachable(ctx)
		return HealthResultMsg{OK: ok}
	}
}

func (m *Model) tickCmd() tea.Cmd {
	return tea.Tick(healthInterval, func(_ time.Time) tea.Msg {
		return tickMsg{}
	})
}

// ── Rendering helpers ──────────────────────────────────────────────────────

func (m *Model) header() string {
	parts := []string{"marrow", screenNames[m.Screen]}
	if !m.EngramOK {
		parts = append(parts, "[DEGRADED]")
	}
	return strings.Join(parts, " — ")
}

func (m *Model) footer() string {
	var keys string
	switch m.Screen {
	case ScreenDashboard:
		keys = "↑/↓:navigate  enter:list  a:add  q:quit"
	case ScreenList:
		keys = "↑/↓:navigate  enter:triage  /:search  esc:back  q:quit"
	case ScreenAdd:
		if m.AddStep == 0 {
			keys = "type URL  enter:next  esc:cancel  q:quit"
		} else if m.AddStep == 1 {
			keys = "type title  enter:submit  esc:cancel  q:quit"
		} else {
			keys = "submitting..."
		}
	case ScreenTriage:
		keys = "↑/↓:select bucket  enter:move  esc:back  q:quit"
	case ScreenConfig:
		keys = "↑/↓:navigate  esc:back  q:quit"
	default:
		keys = "q:quit"
	}

	if m.StatusMsg != "" {
		return keys + "  |  " + m.StatusMsg
	}
	return keys
}
