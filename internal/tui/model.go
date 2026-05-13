package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Nick-2455/marrow/internal/app"
	"github.com/Nick-2455/marrow/internal/domain"
)

// Screen identifies which screen is active.
type Screen int

const (
	ScreenDashboard Screen = iota
	ScreenList
	ScreenAdd
	ScreenTriage
	ScreenConfig
)

var screenNames = map[Screen]string{
	ScreenDashboard: "Dashboard",
	ScreenList:      "List",
	ScreenAdd:       "Add",
	ScreenTriage:    "Triage",
	ScreenConfig:    "Config",
}

const (
	sepLine        = "────────────────────────────────────────────────────"
	cursorPrefix   = "> "
	maxDashItems   = 5
	maxTitleLen    = 50
	maxURLLen      = 55
	healthInterval = 30 * time.Second
)

// Model is the root Bubble Tea model — single struct, no sub-models.
type Model struct {
	screen   Screen
	previous Screen
	cursor   int

	// Real data
	resources []domain.Resource
	config    domain.Config
	engramOK  bool
	statusMsg string

	// Add screen state
	addURL   string
	addTitle string
	addStep  int // 0=url, 1=title

	// List screen state
	searchQuery  string
	searching    bool
	bucketFilter domain.Bucket

	// Triage screen state
	triageResource domain.Resource
	bucketCursor   int
	triageMoving   bool
	triageDone     bool
	triageErr      string

	// Dependencies
	deps *app.Deps
}

// NewModel creates a new TUI model.
func NewModel(deps *app.Deps) tea.Model {
	cfg, _ := deps.Loader.Load()
	return &Model{
		screen:   ScreenDashboard,
		previous: ScreenDashboard,
		engramOK: true,
		config:   cfg,
		deps:     deps,
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
		m.engramOK = msg.OK
		if !msg.OK {
			m.statusMsg = "Engram unreachable — using cached data"
		}

	case ResourceLoadedMsg:
		m.resources = msg.Resources
		if msg.Err != "" {
			m.statusMsg = msg.Err
		}

	case AddSubmitMsg:
		m.addStep = 0
		if msg.Err != "" {
			m.statusMsg = "Error: " + msg.Err
		} else {
			m.statusMsg = "Resource added"
			m.addURL = ""
			m.addTitle = ""
			// Refresh resources after add
			return m, m.refreshResourcesCmd()
		}

	case TriageMoveMsg:
		m.triageMoving = false
		if msg.Err != "" {
			m.triageErr = msg.Err
			m.triageDone = false
			m.statusMsg = "Error: " + msg.Err
		} else {
			m.triageDone = true
			m.statusMsg = "Resource moved to " + string(msg.Bucket)
			// Refresh resources after move
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

	switch m.screen {
	case ScreenDashboard:
		b.WriteString(m.viewDashboard())
	case ScreenList:
		b.WriteString(m.viewList())
	case ScreenAdd:
		b.WriteString(m.viewAdd())
	case ScreenTriage:
		b.WriteString(m.viewTriage())
	case ScreenConfig:
		b.WriteString(m.viewConfig())
	}

	b.WriteString("\n")
	b.WriteString(sepLine)
	b.WriteString("\n")
	b.WriteString(m.footer())

	return b.String()
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
			if m.screen != ScreenDashboard {
				m.previous = m.screen
				m.screen = ScreenDashboard
				m.cursor = 0
				return m, m.refreshResourcesCmd()
			}
		case "a":
			if m.screen != ScreenAdd {
				m.previous = m.screen
				m.screen = ScreenAdd
				m.addStep = 0
				m.addURL = ""
				m.addTitle = ""
				return m, nil
			}
		case "t":
			if m.screen != ScreenTriage && len(m.resources) > 0 {
				m.previous = m.screen
				m.screen = ScreenTriage
				m.cursor = 0
				m.triageDone = false
				m.triageErr = ""
				m.triageMoving = false
				return m, nil
			}
		case "c":
			if m.screen != ScreenConfig {
				m.previous = m.screen
				m.screen = ScreenConfig
				m.cursor = 0
				return m, nil
			}
		}
	}

	// ESC: go back
	if msg.Type == tea.KeyEscape {
		return m.handleEscape()
	}

	// Screen-specific handling
	switch m.screen {
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
	switch m.screen {
	case ScreenList:
		m.screen = ScreenDashboard
		m.cursor = 0
		m.bucketFilter = ""
		m.searchQuery = ""
		m.searching = false
		return m, nil
	case ScreenAdd:
		m.screen = ScreenDashboard
		m.addStep = 0
		m.addURL = ""
		m.addTitle = ""
		return m, nil
	case ScreenTriage:
		if m.triageMoving {
			return m, nil // don't escape while moving
		}
		m.screen = m.previous
		if m.screen == ScreenTriage || m.screen == ScreenAdd {
			m.screen = ScreenDashboard
		}
		m.cursor = 0
		m.triageDone = false
		m.triageErr = ""
		return m, nil
	case ScreenConfig:
		m.screen = ScreenDashboard
		m.cursor = 0
		return m, nil
	default:
		return m, nil
	}
}

func (m *Model) isInputFocused() bool {
	return m.screen == ScreenAdd
}

// ── Dashboard ──────────────────────────────────────────────────────────────

func (m *Model) viewDashboard() string {
	var b strings.Builder

	b.WriteString("marrow — Dashboard\n\n")

	// Group by bucket
	buckets := make(map[domain.Bucket][]domain.Resource)
	for _, r := range m.resources {
		buckets[r.Bucket] = append(buckets[r.Bucket], r)
	}

	for i, bucket := range domain.AllBuckets() {
		items := buckets[bucket]
		count := len(items)

		prefix := "  "
		if i == m.cursor {
			prefix = cursorPrefix
		}
		b.WriteString(fmt.Sprintf("%s%s (%d)\n", prefix, bucket, count))

		if count == 0 {
			b.WriteString("  (empty)\n")
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
				b.WriteString(fmt.Sprintf("  • %s\n", truncate(title, maxTitleLen)))
			}
			if count > maxDashItems {
				b.WriteString(fmt.Sprintf("  ... and %d more\n", count-maxDashItems))
			}
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (m *Model) handleDashboardKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < 3 {
			m.cursor++
		}
	case "enter":
		buckets := domain.AllBuckets()
		if m.cursor < len(buckets) {
			m.bucketFilter = buckets[m.cursor]
			m.previous = ScreenDashboard
			m.screen = ScreenList
			m.cursor = 0
		}
	}
	return nil
}

// ── List ───────────────────────────────────────────────────────────────────

func (m *Model) viewList() string {
	var b strings.Builder

	filtered := m.filteredResources()

	if m.searching {
		b.WriteString(fmt.Sprintf("/ %s\n\n", m.searchQuery))
	}

	if m.bucketFilter != "" {
		b.WriteString(fmt.Sprintf("Filter: %s\n\n", m.bucketFilter))
	}

	if len(filtered) == 0 {
		b.WriteString("No resources found\n")
		return b.String()
	}

	for i, r := range filtered {
		prefix := "  "
		if i == m.cursor {
			prefix = cursorPrefix
		}
		title := truncate(r.Title, maxTitleLen)
		if title == "" {
			title = truncate(r.URL, maxTitleLen)
		}
		bucket := string(r.Bucket)
		b.WriteString(fmt.Sprintf("%s%s [%s]\n", prefix, title, bucket))
	}

	return b.String()
}

func (m *Model) handleListKey(msg tea.KeyMsg) tea.Cmd {
	if m.searching {
		return m.handleListSearch(msg)
	}

	switch msg.String() {
	case "/":
		m.searching = true
		m.searchQuery = ""
		return nil
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		filtered := m.filteredResources()
		if m.cursor < len(filtered)-1 {
			m.cursor++
		}
	case "enter":
		filtered := m.filteredResources()
		if len(filtered) > 0 && m.cursor < len(filtered) {
			m.triageResource = filtered[m.cursor]
			m.previous = ScreenList
			m.screen = ScreenTriage
			m.bucketCursor = 0
			m.triageDone = false
			m.triageErr = ""
			m.triageMoving = false
		}
	}
	return nil
}

func (m *Model) handleListSearch(msg tea.KeyMsg) tea.Cmd {
	switch msg.Type {
	case tea.KeyEnter, tea.KeyEscape:
		m.searching = false
		m.cursor = 0
	case tea.KeyBackspace:
		if len(m.searchQuery) > 0 {
			m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
		}
	default:
		if msg.Type == tea.KeyRunes {
			m.searchQuery += string(msg.Runes)
		}
	}
	return nil
}

func (m *Model) filteredResources() []domain.Resource {
	filtered := m.resources

	if m.bucketFilter != "" {
		var out []domain.Resource
		for _, r := range filtered {
			if r.Bucket == m.bucketFilter {
				out = append(out, r)
			}
		}
		filtered = out
	}

	if m.searchQuery != "" {
		q := strings.ToLower(m.searchQuery)
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

// ── Add ────────────────────────────────────────────────────────────────────

func (m *Model) viewAdd() string {
	var b strings.Builder

	b.WriteString("Add Resource\n\n")

	// URL field
	if m.addStep == 0 {
		b.WriteString(fmt.Sprintf("URL: %s▌\n", m.addURL))
	} else {
		b.WriteString(fmt.Sprintf("URL: %s\n", m.addURL))
	}

	// Title field (only after URL step)
	if m.addStep >= 1 {
		if m.addStep == 1 {
			b.WriteString(fmt.Sprintf("Title: %s▌\n", m.addTitle))
		} else {
			b.WriteString(fmt.Sprintf("Title: %s\n", m.addTitle))
		}
	}

	return b.String()
}

func (m *Model) handleAddKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.Type {
	case tea.KeyEnter:
		if m.addStep == 0 {
			if m.addURL == "" {
				return nil
			}
			m.addStep = 1
			return nil
		}
		if m.addStep == 1 {
			m.addStep = 2 // submitting
			return m.submitAddCmd()
		}
	case tea.KeyBackspace:
		if m.addStep == 0 && len(m.addURL) > 0 {
			m.addURL = m.addURL[:len(m.addURL)-1]
		} else if m.addStep == 1 && len(m.addTitle) > 0 {
			m.addTitle = m.addTitle[:len(m.addTitle)-1]
		}
	case tea.KeyRunes:
		if m.addStep == 0 {
			m.addURL += string(msg.Runes)
		} else if m.addStep == 1 {
			m.addTitle += string(msg.Runes)
		}
	}
	return nil
}

func (m *Model) submitAddCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		resource := domain.Resource{
			URL:    m.addURL,
			Title:  m.addTitle,
			Bucket: domain.BucketInbox,
		}

		id, err := m.deps.Engram.CreateResource(ctx, resource)
		if err != nil {
			return AddSubmitMsg{Err: err.Error()}
		}

		pos := domain.TriagePosition{
			ResourceID: id,
			Bucket:     domain.BucketInbox,
		}
		if err := m.deps.Store.SetTriagePosition(ctx, pos); err != nil {
			return AddSubmitMsg{Err: err.Error()}
		}

		_ = m.deps.Store.InvalidateSearchCache(ctx)

		return AddSubmitMsg{ID: id}
	}
}

// ── Triage ─────────────────────────────────────────────────────────────────

func (m *Model) viewTriage() string {
	var b strings.Builder

	if m.triageMoving {
		b.WriteString("Moving resource...\n")
		return b.String()
	}

	if m.triageDone {
		b.WriteString(fmt.Sprintf("Moved to %s\n", m.triageResource.Bucket))
		return b.String()
	}

	if m.triageErr != "" {
		b.WriteString(fmt.Sprintf("Error: %s\n\n", m.triageErr))
	}

	title := m.triageResource.Title
	if title == "" {
		title = m.triageResource.URL
	}
	b.WriteString(fmt.Sprintf("Move: %s\n\n", truncate(title, maxTitleLen)))

	buckets := domain.AllBuckets()
	for i, bucket := range buckets {
		prefix := "  "
		if i == m.bucketCursor {
			prefix = cursorPrefix
		}
		b.WriteString(fmt.Sprintf("%s%s\n", prefix, bucket))
	}

	return b.String()
}

func (m *Model) handleTriageKey(msg tea.KeyMsg) tea.Cmd {
	if m.triageMoving || m.triageDone {
		return nil
	}

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
		m.triageMoving = true
		return m.moveResourceCmd(buckets[m.bucketCursor])
	}

	return nil
}

func (m *Model) moveResourceCmd(target domain.Bucket) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		r := m.triageResource

		// Save old position for rollback
		oldPos, err := m.deps.Store.GetTriagePosition(ctx, r.ID)
		if err != nil && err != domain.ErrTriageNotFound {
			return TriageMoveMsg{Err: err.Error()}
		}

		// Update SQLite first
		newPos := domain.TriagePosition{
			ResourceID: r.ID,
			Bucket:     target,
		}
		if err := m.deps.Store.SetTriagePosition(ctx, newPos); err != nil {
			return TriageMoveMsg{Err: err.Error()}
		}

		// Update Engram
		if err := m.deps.Engram.UpdateResource(ctx, r.ID, map[string]any{
			"bucket": string(target),
		}); err != nil {
			// Rollback SQLite
			if oldPos.ResourceID != "" {
				_ = m.deps.Store.SetTriagePosition(ctx, oldPos)
			}
			return TriageMoveMsg{Err: err.Error()}
		}

		return TriageMoveMsg{Bucket: target}
	}
}

// ── Config ─────────────────────────────────────────────────────────────────

func (m *Model) viewConfig() string {
	var b strings.Builder

	b.WriteString("Configuration\n\n")

	fields := []struct {
		name  string
		value string
		mask  bool
	}{
		{"Profile", m.config.Profile, false},
		{"Engram Path", m.config.EngramPath, false},
		{"Triage Model", m.config.ModelPrefs.Triage, false},
		{"Summary Model", m.config.ModelPrefs.Summary, false},
	}

	for i, f := range fields {
		prefix := "  "
		if i == m.cursor {
			prefix = cursorPrefix
		}
		val := f.value
		if f.mask && val != "" {
			val = maskKey(val)
		}
		b.WriteString(fmt.Sprintf("%s%s: %s\n", prefix, f.name, val))
	}

	b.WriteString("\n(read-only)\n")

	return b.String()
}

func (m *Model) handleConfigKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < 3 {
			m.cursor++
		}
	}
	return nil
}

// ── Commands ───────────────────────────────────────────────────────────────

func (m *Model) refreshResourcesCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		positions, err := m.deps.Store.GetAllTriagePositions(ctx)
		if err != nil {
			return ResourceLoadedMsg{Err: fmt.Sprintf("load resources: %v", err)}
		}

		resources := make([]domain.Resource, 0, len(positions))
		for _, pos := range positions {
			r := domain.Resource{
				ID:     pos.ResourceID,
				Bucket: pos.Bucket,
			}
			full, err := m.deps.Engram.GetResource(ctx, pos.ResourceID)
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
		ok := m.deps.Engram.IsReachable(ctx)
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
	parts := []string{"marrow", screenNames[m.screen]}
	if !m.engramOK {
		parts = append(parts, "[DEGRADED]")
	}
	return strings.Join(parts, " — ")
}

func (m *Model) footer() string {
	var keys string
	switch m.screen {
	case ScreenDashboard:
		keys = "↑/↓:navigate  enter:list  a:add  q:quit"
	case ScreenList:
		keys = "↑/↓:navigate  enter:triage  /:search  esc:back  q:quit"
	case ScreenAdd:
		if m.addStep == 0 {
			keys = "type URL  enter:next  esc:cancel  q:quit"
		} else if m.addStep == 1 {
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

	if m.statusMsg != "" {
		return keys + "  |  " + m.statusMsg
	}
	return keys
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func maskKey(s string) string {
	if len(s) <= 8 {
		return "****"
	}
	return s[:4] + "****" + s[len(s)-4:]
}
