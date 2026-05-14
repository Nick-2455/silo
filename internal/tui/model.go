package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Nick-2455/marrow/internal/app"
	"github.com/Nick-2455/marrow/internal/domain"
	"github.com/Nick-2455/marrow/internal/obsidian"
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

	// Taxonomy screen state
	DomainTree       []domain.DomainWithSubareas
	DomainTreeCursor int

	// Projects screen state
	Projects        []domain.Project
	ProjectCursor   int
	SelectedProject *domain.Project
	ProjectSubareas []domain.Subarea

	// Sessions screen state
	Sessions         []domain.Session
	SessionCursor    int
	SelectedSession  *domain.Session
	SessionLearnings []domain.Learning

	// Learnings screen state
	Learnings     []domain.Learning
	LearningCursor int

	// Obsidian sync state
	EditingVaultPath bool
	SyncVaultPath    string

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
		m.seedDemoDataCmd(),
	)
}

// seedDemoDataCmd seeds demo domains, subareas, and projects if the graph is empty.
// TODO: Remove once MCP tools handle taxonomy/project creation (PR 3).
func (m *Model) seedDemoDataCmd() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		store := m.Deps.GraphStore

		// Only seed if no domains exist
		domains, err := store.ListNodesByType(ctx, domain.NodeTypeDomain)
		if err != nil || len(domains) > 0 {
			return nil
		}

		// Dev domain
		devID := "domain/dev"
		_ = store.UpsertNode(ctx, domain.GraphNode{EngramID: devID, NodeType: domain.NodeTypeDomain, Title: "Dev", Active: true})
		_ = store.UpsertNode(ctx, domain.GraphNode{EngramID: "subarea/dev/backend", NodeType: domain.NodeTypeSubarea, Title: "Backend", Active: true})
		_ = store.AddEdge(ctx, devID, "subarea/dev/backend", domain.EdgeContains)
		_ = store.UpsertNode(ctx, domain.GraphNode{EngramID: "subarea/dev/ios", NodeType: domain.NodeTypeSubarea, Title: "iOS", Active: true})
		_ = store.AddEdge(ctx, devID, "subarea/dev/ios", domain.EdgeContains)

		// Filosofía domain
		filoID := "domain/filosofia"
		_ = store.UpsertNode(ctx, domain.GraphNode{EngramID: filoID, NodeType: domain.NodeTypeDomain, Title: "Filosofía", Active: true})
		_ = store.UpsertNode(ctx, domain.GraphNode{EngramID: "subarea/filosofia/estoicismo", NodeType: domain.NodeTypeSubarea, Title: "Estoicismo", Active: true})
		_ = store.AddEdge(ctx, filoID, "subarea/filosofia/estoicismo", domain.EdgeContains)

		// Projects
		_ = store.UpsertNode(ctx, domain.GraphNode{EngramID: "project/marrow", NodeType: domain.NodeTypeProject, Title: "marrow", Active: true})
		_ = store.AddEdge(ctx, "project/marrow", "subarea/dev/backend", domain.EdgeAppliesTo)

		_ = store.UpsertNode(ctx, domain.GraphNode{EngramID: "project/kitting-inspection", NodeType: domain.NodeTypeProject, Title: "kitting-inspection", Active: true})
		_ = store.AddEdge(ctx, "project/kitting-inspection", "subarea/dev/backend", domain.EdgeAppliesTo)

		_ = store.UpsertNode(ctx, domain.GraphNode{EngramID: "project/publora", NodeType: domain.NodeTypeProject, Title: "publora", Active: false})
		_ = store.AddEdge(ctx, "project/publora", "subarea/dev/ios", domain.EdgeAppliesTo)

		// Sessions
		s1ID := "session/debug-engram-client"
		_ = store.UpsertNode(ctx, domain.GraphNode{EngramID: s1ID, NodeType: domain.NodeTypeSession, Title: "Debug de Engram MCP client", Active: true})
		_ = store.AddEdge(ctx, s1ID, "project/marrow", domain.EdgeWorkedOn)

		s2ID := "session/refactor-tui"
		_ = store.UpsertNode(ctx, domain.GraphNode{EngramID: s2ID, NodeType: domain.NodeTypeSession, Title: "Refactor de TUI a arquitectura Gentle AI", Active: true})
		_ = store.AddEdge(ctx, s2ID, "project/marrow", domain.EdgeWorkedOn)

		// Learnings
		l1ID := "learning/mem-update-replaces"
		_ = store.UpsertNode(ctx, domain.GraphNode{EngramID: l1ID, NodeType: domain.NodeTypeLearning, Title: "mem_update reemplaza el contenido entero — no mergea", Active: true})
		_ = store.AddEdge(ctx, l1ID, s1ID, domain.EdgeLearnedFrom)
		_ = store.AddEdge(ctx, l1ID, "subarea/dev/backend", domain.EdgeAppliesTo)
		_ = store.AddEdge(ctx, l1ID, "project/marrow", domain.EdgeAppliesTo)

		l2ID := "learning/engram-json-ids"
		_ = store.UpsertNode(ctx, domain.GraphNode{EngramID: l2ID, NodeType: domain.NodeTypeLearning, Title: "Engram responde JSON con id numérico, no texto con #", Active: true})
		_ = store.AddEdge(ctx, l2ID, s1ID, domain.EdgeLearnedFrom)
		_ = store.AddEdge(ctx, l2ID, "subarea/dev/backend", domain.EdgeAppliesTo)
		_ = store.AddEdge(ctx, l2ID, "project/marrow", domain.EdgeAppliesTo)

		l3ID := "learning/screen-router-pattern"
		_ = store.UpsertNode(ctx, domain.GraphNode{EngramID: l3ID, NodeType: domain.NodeTypeLearning, Title: "Patrón Screen+Router separa rendering de lógica de navegación", Active: true})
		_ = store.AddEdge(ctx, l3ID, s2ID, domain.EdgeLearnedFrom)
		_ = store.AddEdge(ctx, l3ID, "subarea/dev/backend", domain.EdgeAppliesTo)
		_ = store.AddEdge(ctx, l3ID, "subarea/dev/ios", domain.EdgeAppliesTo)
		_ = store.AddEdge(ctx, l3ID, "project/marrow", domain.EdgeAppliesTo)

		return nil
	}
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

	case SyncDoneMsg:
		if msg.Err != "" {
			m.StatusMsg = "Sync error: " + msg.Err
		} else if msg.Report != nil {
			m.StatusMsg = fmt.Sprintf("Synced %d nodes, %d edges to Obsidian", msg.Report.NodesWritten, msg.Report.EdgesWritten)
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

	// Vault path prompt (shown over any screen when editing)
	if m.EditingVaultPath {
		b.WriteString("Obsidian vault path:\n")
		b.WriteString(fmt.Sprintf("  %s▌\n\n", m.SyncVaultPath))
	}

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
	case ScreenDomainTree:
		b.WriteString(screens.RenderDomainTree(m.DomainTree, m.DomainTreeCursor))
	case ScreenProjects:
		b.WriteString(screens.RenderProjectList(m.Projects, m.ProjectCursor))
	case ScreenProjectDetail:
		b.WriteString(screens.RenderProjectDetail(*m.SelectedProject, m.ProjectSubareas, m.Cursor))
	case ScreenSessions:
		b.WriteString(screens.RenderSessionList(m.Sessions, m.SessionCursor))
	case ScreenSessionDetail:
		b.WriteString(screens.RenderSessionDetail(*m.SelectedSession, m.SessionLearnings, m.Cursor))
	case ScreenLearnings:
		b.WriteString(screens.RenderLearningList(m.Learnings, m.LearningCursor))
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

	// Vault path input mode
	if m.EditingVaultPath {
		return m.handleVaultPathInput(msg)
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
		case "g":
			if m.Screen != ScreenDomainTree {
				m.setScreen(ScreenDomainTree)
				m.loadDomainTree()
				return m, nil
			}
		case "p":
			if m.Screen != ScreenProjects && m.Screen != ScreenProjectDetail {
				m.setScreen(ScreenProjects)
				m.loadProjects()
				return m, nil
			}
		case "s":
			if m.Screen != ScreenSessions && m.Screen != ScreenSessionDetail {
				m.setScreen(ScreenSessions)
				m.loadSessions()
				return m, nil
			}
		case "l":
			if m.Screen != ScreenLearnings {
				m.setScreen(ScreenLearnings)
				m.loadLearnings()
				return m, nil
			}
		case "o":
			if m.Screen == ScreenDashboard {
				if m.Config.ObsidianVaultPath != "" {
					m.StatusMsg = "Syncing to Obsidian..."
					return m, m.syncObsidianCmd()
				}
				// No vault path yet — ask interactively
				m.EditingVaultPath = true
				m.SyncVaultPath = ""
				m.StatusMsg = "Enter Obsidian vault path and press Enter"
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
	case ScreenDomainTree:
		return m, m.handleDomainTreeKey(msg)
	case ScreenProjects:
		return m, m.handleProjectsKey(msg)
	case ScreenProjectDetail:
		return m, m.handleProjectDetailKey(msg)
	case ScreenSessions:
		return m, m.handleSessionsKey(msg)
	case ScreenSessionDetail:
		return m, m.handleSessionDetailKey(msg)
	case ScreenLearnings:
		return m, m.handleLearningsKey(msg)
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
	case ScreenDomainTree:
		m.Screen = ScreenDashboard
		m.DomainTreeCursor = 0
		return m, nil
	case ScreenProjects:
		m.Screen = ScreenDashboard
		m.ProjectCursor = 0
		return m, nil
	case ScreenProjectDetail:
		m.Screen = ScreenProjects
		m.Cursor = 0
		return m, nil
	case ScreenSessions:
		m.Screen = ScreenDashboard
		m.SessionCursor = 0
		return m, nil
	case ScreenSessionDetail:
		m.Screen = ScreenSessions
		m.Cursor = 0
		return m, nil
	case ScreenLearnings:
		m.Screen = ScreenDashboard
		m.LearningCursor = 0
		return m, nil
	default:
		return m, nil
	}
}

func (m *Model) isInputFocused() bool {
	return m.Screen == ScreenAdd || m.EditingVaultPath
}

func (m *Model) handleVaultPathInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		if m.SyncVaultPath != "" {
			m.EditingVaultPath = false
			m.Config.ObsidianVaultPath = m.SyncVaultPath
			_ = m.Deps.Loader.Save(m.Config)
			m.StatusMsg = "Syncing to Obsidian..."
			return m, m.syncObsidianCmd()
		}
		return m, nil
	case tea.KeyEscape:
		m.EditingVaultPath = false
		m.SyncVaultPath = ""
		m.StatusMsg = ""
		return m, nil
	case tea.KeyBackspace:
		if len(m.SyncVaultPath) > 0 {
			m.SyncVaultPath = m.SyncVaultPath[:len(m.SyncVaultPath)-1]
		}
	case tea.KeyRunes:
		m.SyncVaultPath += string(msg.Runes)
	}
	return m, nil
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

// ── Domain Tree ────────────────────────────────────────────────────────────

func (m *Model) handleDomainTreeKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "up", "k":
		if m.DomainTreeCursor > 0 {
			m.DomainTreeCursor--
		}
	case "down", "j":
		if m.DomainTreeCursor < len(m.DomainTree)-1 {
			m.DomainTreeCursor++
		}
	case "enter":
		// Read-only for now — future: expand/collapse domain
	}
	return nil
}

func (m *Model) loadDomainTree() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tree, err := m.Deps.GraphStore.GetDomainTree(ctx)
	if err != nil {
		m.StatusMsg = fmt.Sprintf("load taxonomy: %v", err)
		return
	}
	m.DomainTree = tree
	m.DomainTreeCursor = 0
}

// ── Projects ───────────────────────────────────────────────────────────────

func (m *Model) handleProjectsKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "up", "k":
		if m.ProjectCursor > 0 {
			m.ProjectCursor--
		}
	case "down", "j":
		if m.ProjectCursor < len(m.Projects)-1 {
			m.ProjectCursor++
		}
	case "enter":
		if len(m.Projects) > 0 && m.ProjectCursor < len(m.Projects) {
			m.SelectedProject = &m.Projects[m.ProjectCursor]
			m.setScreen(ScreenProjectDetail)
			m.loadProjectSubareas()
		}
	}
	return nil
}

func (m *Model) loadProjects() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	projects, err := m.Deps.GraphStore.ListActiveProjects(ctx)
	if err != nil {
		m.StatusMsg = fmt.Sprintf("load projects: %v", err)
		return
	}
	m.Projects = projects
	m.ProjectCursor = 0
	m.SelectedProject = nil
}

func (m *Model) loadProjectSubareas() {
	if m.SelectedProject == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	subareas := make([]domain.Subarea, 0, len(m.SelectedProject.SubareaIDs))
	for _, id := range m.SelectedProject.SubareaIDs {
		node, err := m.Deps.GraphStore.GetNode(ctx, id)
		if err != nil {
			continue
		}
		subareas = append(subareas, domain.Subarea{
			Name:   node.Title,
			Active: node.Active,
		})
	}
	m.ProjectSubareas = subareas
	m.Cursor = 0
}

// ── Project Detail ─────────────────────────────────────────────────────────

func (m *Model) handleProjectDetailKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "up", "k":
		if m.Cursor > 0 {
			m.Cursor--
		}
	case "down", "j":
		if m.Cursor < len(m.ProjectSubareas)-1 {
			m.Cursor++
		}
	}
	return nil
}

// ── Sessions ───────────────────────────────────────────────────────────────

func (m *Model) handleSessionsKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "up", "k":
		if m.SessionCursor > 0 {
			m.SessionCursor--
		}
	case "down", "j":
		if m.SessionCursor < len(m.Sessions)-1 {
			m.SessionCursor++
		}
	case "enter":
		if len(m.Sessions) > 0 && m.SessionCursor < len(m.Sessions) {
			m.SelectedSession = &m.Sessions[m.SessionCursor]
			m.setScreen(ScreenSessionDetail)
			m.loadSessionLearnings()
		}
	}
	return nil
}

func (m *Model) handleSessionDetailKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "up", "k":
		if m.Cursor > 0 {
			m.Cursor--
		}
	case "down", "j":
		if m.Cursor < len(m.SessionLearnings)-1 {
			m.Cursor++
		}
	}
	return nil
}

func (m *Model) loadSessions() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Load sessions from all projects
	projects, err := m.Deps.GraphStore.ListActiveProjects(ctx)
	if err != nil {
		m.StatusMsg = fmt.Sprintf("load projects: %v", err)
		return
	}

	var allSessions []domain.Session
	for _, p := range projects {
		// Use the project's engram ID — reconstruct from slug
		projectNodes, err := m.Deps.GraphStore.ListNodesByType(ctx, domain.NodeTypeProject)
		if err != nil {
			continue
		}
		for _, pn := range projectNodes {
			if domain.Slugify(pn.Title) == p.Slug {
				sessions, err := m.Deps.GraphStore.ListSessions(ctx, pn.EngramID)
				if err == nil {
					allSessions = append(allSessions, sessions...)
				}
				break
			}
		}
	}

	m.Sessions = allSessions
	m.SessionCursor = 0
	m.SelectedSession = nil
}

func (m *Model) loadSessionLearnings() {
	if m.SelectedSession == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Find learnings linked to this session via learned_from edge
	learnings, err := m.Deps.GraphStore.ListLearnings(ctx, "")
	if err != nil {
		m.StatusMsg = fmt.Sprintf("load learnings: %v", err)
		return
	}

	var filtered []domain.Learning
	for _, l := range learnings {
		if l.SessionID == m.SelectedSession.ID {
			filtered = append(filtered, l)
		}
	}

	m.SessionLearnings = filtered
	m.Cursor = 0
}

// ── Learnings ──────────────────────────────────────────────────────────────

func (m *Model) handleLearningsKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "up", "k":
		if m.LearningCursor > 0 {
			m.LearningCursor--
		}
	case "down", "j":
		if m.LearningCursor < len(m.Learnings)-1 {
			m.LearningCursor++
		}
	}
	return nil
}

func (m *Model) loadLearnings() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	learnings, err := m.Deps.GraphStore.ListLearnings(ctx, "")
	if err != nil {
		m.StatusMsg = fmt.Sprintf("load learnings: %v", err)
		return
	}

	m.Learnings = learnings
	m.LearningCursor = 0
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

func (m *Model) syncObsidianCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		syncer := obsidian.Syncer{
			Store:     m.Deps.GraphStore,
			VaultPath: m.Config.ObsidianVaultPath,
		}
		report, err := syncer.SyncAll(ctx)
		if err != nil {
			return SyncDoneMsg{Err: err.Error()}
		}
		return SyncDoneMsg{Report: report}
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
		keys = "↑/↓:navigate  enter:list  a:add  g:domains  p:projects  s:sessions  l:learnings  o:sync  q:quit"
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
	case ScreenDomainTree:
		keys = "↑/↓:navigate  esc:back  q:quit"
	case ScreenProjects:
		keys = "↑/↓:navigate  enter:details  esc:back  q:quit"
	case ScreenProjectDetail:
		keys = "↑/↓:navigate  esc:back  q:quit"
	case ScreenSessions:
		keys = "↑/↓:navigate  enter:details  esc:back  q:quit"
	case ScreenSessionDetail:
		keys = "↑/↓:navigate  esc:back  q:quit"
	case ScreenLearnings:
		keys = "↑/↓:navigate  esc:back  q:quit"
	default:
		keys = "q:quit"
	}

	if m.StatusMsg != "" {
		return keys + "  |  " + m.StatusMsg
	}
	return keys
}
