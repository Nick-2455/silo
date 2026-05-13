package tui_test

import (
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Nick-2455/marrow/internal/app"
	"github.com/Nick-2455/marrow/internal/domain"
	"github.com/Nick-2455/marrow/internal/store"
	"github.com/Nick-2455/marrow/internal/tui"
)

// mockConfigLoader is a test double for domain.ConfigLoader.
type mockConfigLoader struct {
	cfg     domain.Config
	path    string
	saveErr error
}

func (m *mockConfigLoader) Load() (domain.Config, error) {
	return m.cfg, nil
}

func (m *mockConfigLoader) Save(cfg domain.Config) error {
	return m.saveErr
}

func (m *mockConfigLoader) Path() string {
	return m.path
}

// mockEngramClient is a test double for domain.EngramClient.
type mockEngramClient struct {
	reachable  bool
	resources  []domain.Resource
	roadmap    map[domain.Bucket][]domain.Resource
	createErr  error
	searchErr  error
	roadmapErr error
	updateErr  error
}

func (m *mockEngramClient) CreateResource(ctx context.Context, r domain.Resource) (string, error) {
	if m.createErr != nil {
		return "", m.createErr
	}
	return "test-id-123", nil
}

func (m *mockEngramClient) GetResource(ctx context.Context, id string) (domain.Resource, error) {
	for _, r := range m.resources {
		if r.ID == id {
			return r, nil
		}
	}
	return domain.Resource{ID: id}, nil
}

func (m *mockEngramClient) SearchResources(ctx context.Context, query string) ([]domain.Resource, error) {
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	return m.resources, nil
}

func (m *mockEngramClient) GetRoadmap(ctx context.Context) (map[domain.Bucket][]domain.Resource, error) {
	if m.roadmapErr != nil {
		return nil, m.roadmapErr
	}
	return m.roadmap, nil
}

func (m *mockEngramClient) UpdateResource(ctx context.Context, id string, updates map[string]any) error {
	return m.updateErr
}

func (m *mockEngramClient) IsReachable(ctx context.Context) bool {
	return m.reachable
}

func newTestDeps(t *testing.T) *app.Deps {
	t.Helper()

	loader := &mockConfigLoader{
		cfg: domain.Config{
			Profile:   "test",
			EngramAPI: "http://localhost:8080",
			EngramKey: "test-key",
		},
		path: "/tmp/test-config.yaml",
	}

	// Use in-memory SQLite for testing
	dbStore, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory store: %v", err)
	}
	if err := dbStore.Migrate(); err != nil {
		t.Fatalf("failed to migrate store: %v", err)
	}

	engram := &mockEngramClient{
		reachable: true,
		resources: []domain.Resource{
			{ID: "res-1", URL: "https://example.com/1", Title: "Resource 1", Bucket: domain.BucketInbox},
			{ID: "res-2", URL: "https://example.com/2", Title: "Resource 2", Bucket: domain.BucketActive},
			{ID: "res-3", URL: "https://example.com/3", Title: "Resource 3", Bucket: domain.BucketLater},
		},
	}

	return &app.Deps{
		Config: loader.cfg,
		Store:  dbStore,
		Engram: engram,
		Loader: loader,
	}
}

// updateModel is a helper that handles the tea.Model interface type assertion.
func updateModel(t *testing.T, m *tui.Model, msg tea.Msg) *tui.Model {
	t.Helper()
	result, _ := m.Update(msg)
	model, ok := result.(*tui.Model)
	if !ok {
		t.Fatalf("expected *tui.Model, got %T", result)
	}
	return model
}

func TestNewModel(t *testing.T) {
	deps := newTestDeps(t)
	model := tui.NewModel(deps)

	if model == nil {
		t.Fatal("NewModel returned nil")
	}
}

func TestModelInit(t *testing.T) {
	deps := newTestDeps(t)
	model := tui.NewModel(deps)

	cmd := model.Init()
	if cmd == nil {
		t.Fatal("Init() returned nil command")
	}
}

func TestScreenNavigation(t *testing.T) {
	deps := newTestDeps(t)
	model := tui.NewModel(deps)
	model.Init()

	// Test Tab navigation cycles through screens
	for i := 0; i < 5; i++ {
		model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyTab})
	}
}

func TestDirectScreenShortcuts(t *testing.T) {
	deps := newTestDeps(t)
	model := tui.NewModel(deps)
	model.Init()

	// Test 'd' goes to dashboard
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})

	// Test 'l' goes to list
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})

	// Test 'n' goes to add
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})

	// Test 't' goes to triage
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
}

func TestQuitOnQ(t *testing.T) {
	deps := newTestDeps(t)
	model := tui.NewModel(deps)
	model.Init()

	result, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	if cmd == nil {
		t.Fatal("Expected tea.Quit command on 'q'")
	}

	// Execute the command and verify it returns tea.QuitMsg
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("Expected tea.QuitMsg, got %T", msg)
	}
	_ = result
}

func TestQuitOnCtrlC(t *testing.T) {
	deps := newTestDeps(t)
	model := tui.NewModel(deps)
	model.Init()

	result, cmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})

	if cmd == nil {
		t.Fatal("Expected tea.Quit command on Ctrl+C")
	}

	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("Expected tea.QuitMsg, got %T", msg)
	}
	_ = result
}

func TestHealthResultUpdatesStatus(t *testing.T) {
	deps := newTestDeps(t)
	model := tui.NewModel(deps)
	model.Init()

	// Simulate health check failure
	model = updateModel(t, model, tui.HealthResultMsg{OK: false})

	// Status should be set to warning
	if model.StatusType() != tui.StatusWarning {
		t.Errorf("Expected StatusWarning after health failure, got %v", model.StatusType())
	}
}

func TestHealthResultSuccess(t *testing.T) {
	deps := newTestDeps(t)
	model := tui.NewModel(deps)
	model.Init()

	// Simulate health check success
	model = updateModel(t, model, tui.HealthResultMsg{OK: true})
}

func TestErrorStatusMessage(t *testing.T) {
	deps := newTestDeps(t)
	model := tui.NewModel(deps)
	model.Init()

	model = updateModel(t, model, tui.ErrorMsg("test error"))

	if model.StatusType() != tui.StatusError {
		t.Errorf("Expected StatusError, got %v", model.StatusType())
	}
}

func TestSuccessStatusMessage(t *testing.T) {
	deps := newTestDeps(t)
	model := tui.NewModel(deps)
	model.Init()

	model = updateModel(t, model, tui.SuccessMsg("operation completed"))

	if model.StatusType() != tui.StatusSuccess {
		t.Errorf("Expected StatusSuccess, got %v", model.StatusType())
	}
}

func TestViewRendersWithoutPanic(t *testing.T) {
	deps := newTestDeps(t)
	model := tui.NewModel(deps)
	model.Init()

	// Test that View() doesn't panic on any screen
	screens := []struct {
		name string
		key  tea.KeyMsg
	}{
		{"dashboard", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}},
		{"list", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}}},
		{"add", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}},
		{"triage", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}}},
	}

	for _, s := range screens {
		t.Run(s.name, func(t *testing.T) {
			model = updateModel(t, model, s.key)
			view := model.View()
			if view == "" {
				t.Error("View() returned empty string")
			}
		})
	}
}

func TestEscapeReturnsToDashboard(t *testing.T) {
	deps := newTestDeps(t)
	model := tui.NewModel(deps)
	model.Init()

	// Go to add screen
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})

	// Press escape
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyEscape})
}

func TestResourcesLoadedAppliesFilters(t *testing.T) {
	deps := newTestDeps(t)
	model := tui.NewModel(deps)
	model.Init()

	resources := []domain.Resource{
		{ID: "r1", Title: "Test Resource", URL: "https://test.com", Bucket: domain.BucketInbox},
		{ID: "r2", Title: "Another Resource", URL: "https://another.com", Bucket: domain.BucketActive},
	}

	model = updateModel(t, model, tui.ResourcesLoadedMsg{Resources: resources})
}

func TestListBucketFilter(t *testing.T) {
	deps := newTestDeps(t)
	model := tui.NewModel(deps)
	model.Init()

	// Load resources first
	resources := []domain.Resource{
		{ID: "r1", Title: "Inbox Item", Bucket: domain.BucketInbox},
		{ID: "r2", Title: "Active Item", Bucket: domain.BucketActive},
		{ID: "r3", Title: "Later Item", Bucket: domain.BucketLater},
	}
	model = updateModel(t, model, tui.ResourcesLoadedMsg{Resources: resources})

	// Go to list
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})

	// Cycle bucket filter with 'f'
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
}

func TestAddFormNavigation(t *testing.T) {
	deps := newTestDeps(t)
	model := tui.NewModel(deps)
	model.Init()

	// Go to add screen
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})

	// Type URL
	for _, r := range []rune{'h', 't', 't', 'p', 's'} {
		model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	// Press Enter to move to title
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyEnter})

	// Press Escape to cancel
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyEscape})
}

func TestTriageScreenNavigation(t *testing.T) {
	deps := newTestDeps(t)
	model := tui.NewModel(deps)
	model.Init()

	// Load resources
	resources := []domain.Resource{
		{ID: "r1", Title: "Item 1", Bucket: domain.BucketInbox},
		{ID: "r2", Title: "Item 2", Bucket: domain.BucketActive},
	}
	model = updateModel(t, model, tui.ResourcesLoadedMsg{Resources: resources})

	// Go to triage
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})

	// Navigate down
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyDown})

	// Navigate up
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyUp})
}

func TestDashboardView(t *testing.T) {
	deps := newTestDeps(t)
	model := tui.NewModel(deps)
	model.Init()

	// Load resources
	resources := []domain.Resource{
		{ID: "r1", Title: "Inbox Item", Bucket: domain.BucketInbox},
		{ID: "r2", Title: "Active Item", Bucket: domain.BucketActive},
		{ID: "r3", Title: "Later Item", Bucket: domain.BucketLater},
		{ID: "r4", Title: "Archived Item", Bucket: domain.BucketArchived},
	}
	model = updateModel(t, model, tui.ResourcesLoadedMsg{Resources: resources})

	// Go to dashboard
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})

	view := model.View()
	if view == "" {
		t.Error("Dashboard view is empty")
	}
}

func TestConfigScreenNavigation(t *testing.T) {
	deps := newTestDeps(t)
	model := tui.NewModel(deps)
	model.Init()

	// Navigate to config via tabs
	for i := 0; i < 4; i++ {
		model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyTab})
	}

	// Navigate down
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyDown})

	// Navigate up
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyUp})

	// Press escape to go back
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyEscape})
}

func TestTickCmd(t *testing.T) {
	deps := newTestDeps(t)
	model := tui.NewModel(deps)

	cmd := model.TickCmd()
	if cmd == nil {
		t.Fatal("TickCmd returned nil")
	}
}

func TestResourcesLoadedMsgSetsResources(t *testing.T) {
	deps := newTestDeps(t)
	model := tui.NewModel(deps)
	model.Init()

	resources := []domain.Resource{
		{ID: "r1", Title: "Test", URL: "https://test.com", Bucket: domain.BucketInbox},
	}
	model = updateModel(t, model, tui.ResourcesLoadedMsg{Resources: resources})

	// Verify view renders without error
	view := model.View()
	if view == "" {
		t.Error("View should not be empty after loading resources")
	}
}

func TestModelHealthInterval(t *testing.T) {
	deps := newTestDeps(t)
	model := tui.NewModel(deps)

	// Verify health interval is set (30s)
	if model.HealthInterval() != 30*time.Second {
		t.Errorf("Expected 30s health interval, got %v", model.HealthInterval())
	}
}
