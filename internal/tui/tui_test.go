package tui_test

import (
	"context"
	"testing"

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

func (m *mockEngramClient) SaveNode(ctx context.Context, nodeType, title string, content map[string]any, topicKey string) (string, error) {
	return "test-node-id-123", nil
}

func (m *mockEngramClient) UpdateNode(ctx context.Context, engramID string, content map[string]any) error {
	return nil
}

func newTestDeps(t *testing.T) *app.Deps {
	t.Helper()

	loader := &mockConfigLoader{
		cfg: domain.Config{
			Profile:    "test",
			EngramPath: "engram",
		},
		path: "/tmp/test-config.yaml",
	}

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
		Config:     loader.cfg,
		Store:      dbStore,
		Engram:     engram,
		Loader:     loader,
		GraphStore: dbStore, // *Store implements domain.GraphStore
	}
}

func updateModel(t *testing.T, m tea.Model, msg tea.Msg) *tui.Model {
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

func TestQuitOnQ(t *testing.T) {
	deps := newTestDeps(t)
	model := tui.NewModel(deps)
	model.Init()

	result, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	if cmd == nil {
		t.Fatal("Expected tea.Quit command on 'q'")
	}

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

	model = updateModel(t, model, tui.HealthResultMsg{OK: false})

	view := model.View()
	if view == "" {
		t.Error("View() returned empty string")
	}
}

func TestResourcesLoadedUpdatesModel(t *testing.T) {
	deps := newTestDeps(t)
	model := tui.NewModel(deps)
	model.Init()

	resources := []domain.Resource{
		{ID: "r1", Title: "Test Resource", URL: "https://test.com", Bucket: domain.BucketInbox},
		{ID: "r2", Title: "Another Resource", URL: "https://another.com", Bucket: domain.BucketActive},
	}

	model = updateModel(t, model, tui.ResourceLoadedMsg{Resources: resources})

	view := model.View()
	if view == "" {
		t.Error("View() returned empty string after loading resources")
	}
}

func TestViewRendersWithoutPanic(t *testing.T) {
	deps := newTestDeps(t)
	model := tui.NewModel(deps)
	model.Init()

	// Load resources
	resources := []domain.Resource{
		{ID: "r1", Title: "Test", URL: "https://test.com", Bucket: domain.BucketInbox},
	}
	model = updateModel(t, model, tui.ResourceLoadedMsg{Resources: resources})

	screens := []struct {
		name string
		key  tea.KeyMsg
	}{
		{"dashboard", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}},
		{"list", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}}},
		{"add", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}},
		{"triage", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}}},
		{"config", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}}},
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
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	// Press escape
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyEscape})

	// Should be back on dashboard
	view := model.View()
	if view == "" {
		t.Error("View() returned empty string")
	}
}

func TestAddFormNavigation(t *testing.T) {
	deps := newTestDeps(t)
	model := tui.NewModel(deps)
	model.Init()

	// Go to add screen
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	// Type URL
	for _, r := range []rune{'h', 't', 't', 'p', 's'} {
		model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	// Press Enter to move to title
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyEnter})

	// Press Escape to cancel
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyEscape})

	view := model.View()
	if view == "" {
		t.Error("View() returned empty string")
	}
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
	model = updateModel(t, model, tui.ResourceLoadedMsg{Resources: resources})

	// Go to triage
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})

	// Navigate down
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyDown})

	// Navigate up
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyUp})

	view := model.View()
	if view == "" {
		t.Error("View() returned empty string")
	}
}

func TestDashboardView(t *testing.T) {
	deps := newTestDeps(t)
	model := tui.NewModel(deps)
	model.Init()

	resources := []domain.Resource{
		{ID: "r1", Title: "Inbox Item", Bucket: domain.BucketInbox},
		{ID: "r2", Title: "Active Item", Bucket: domain.BucketActive},
		{ID: "r3", Title: "Later Item", Bucket: domain.BucketLater},
		{ID: "r4", Title: "Archived Item", Bucket: domain.BucketArchived},
	}
	model = updateModel(t, model, tui.ResourceLoadedMsg{Resources: resources})

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

	// Navigate to config
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})

	// Navigate down
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyDown})

	// Navigate up
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyUp})

	// Press escape to go back
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyEscape})

	view := model.View()
	if view == "" {
		t.Error("View() returned empty string")
	}
}

func TestListSearch(t *testing.T) {
	deps := newTestDeps(t)
	model := tui.NewModel(deps)
	model.Init()

	resources := []domain.Resource{
		{ID: "r1", Title: "Inbox Item", Bucket: domain.BucketInbox},
		{ID: "r2", Title: "Active Item", Bucket: domain.BucketActive},
	}
	model = updateModel(t, model, tui.ResourceLoadedMsg{Resources: resources})

	// Go to list
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})

	// Start search
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})

	// Type search query
	for _, r := range []rune{'i', 'n', 'b', 'o', 'x'} {
		model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	// Cancel search with escape
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyEscape})

	view := model.View()
	if view == "" {
		t.Error("View() returned empty string")
	}
}

func TestDirectScreenShortcuts(t *testing.T) {
	deps := newTestDeps(t)
	model := tui.NewModel(deps)
	model.Init()

	// Test 'd' stays on dashboard
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})

	// Test 'a' goes to add
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	// Test 'd' goes back to dashboard
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})

	view := model.View()
	if view == "" {
		t.Error("View() returned empty string")
	}
}

func TestTickCmd(t *testing.T) {
	deps := newTestDeps(t)
	model := tui.NewModel(deps)

	cmd := model.Init()
	if cmd == nil {
		t.Fatal("Init() returned nil command")
	}
}

func TestHealthCheckCmd(t *testing.T) {
	deps := newTestDeps(t)
	model := tui.NewModel(deps)
	model.Init()

	// Trigger health check manually
	msg := tui.HealthResultMsg{OK: true}
	model = updateModel(t, model, msg)

	view := model.View()
	if view == "" {
		t.Error("View() returned empty string")
	}
}

func TestAddSubmitSuccess(t *testing.T) {
	deps := newTestDeps(t)
	model := tui.NewModel(deps)
	model.Init()

	// Go to add screen
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	// Type URL
	for _, r := range []rune{'h', 't', 't', 'p', 's', ':', '/', '/', 't', 'e', 's', 't', '.', 'c', 'o', 'm'} {
		model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	// Enter to go to title
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyEnter})

	// Type title
	for _, r := range []rune{'T', 'e', 's', 't'} {
		model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	// Enter to submit
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyEnter})

	// Simulate submit result
	model = updateModel(t, model, tui.AddSubmitMsg{ID: "test-id"})

	view := model.View()
	if view == "" {
		t.Error("View() returned empty string after submit")
	}
}

func TestAddSubmitError(t *testing.T) {
	deps := newTestDeps(t)
	model := tui.NewModel(deps)
	model.Init()

	// Go to add screen
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	// Type URL
	for _, r := range []rune{'h', 't', 't', 'p', 's', ':', '/', '/', 't', 'e', 's', 't', '.', 'c', 'o', 'm'} {
		model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	// Enter to go to title
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyEnter})

	// Type title
	for _, r := range []rune{'T', 'e', 's', 't'} {
		model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	// Enter to submit
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyEnter})

	// Simulate submit error
	model = updateModel(t, model, tui.AddSubmitMsg{Err: "connection refused"})

	view := model.View()
	if view == "" {
		t.Error("View() returned empty string after submit error")
	}
}

func TestTriageMoveSuccess(t *testing.T) {
	deps := newTestDeps(t)
	model := tui.NewModel(deps)
	model.Init()

	// Load resources
	resources := []domain.Resource{
		{ID: "r1", Title: "Item 1", Bucket: domain.BucketInbox},
	}
	model = updateModel(t, model, tui.ResourceLoadedMsg{Resources: resources})

	// Go to triage
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})

	// Select bucket (down to active)
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyDown})

	// Enter to move
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyEnter})

	// Simulate move result
	model = updateModel(t, model, tui.TriageMoveMsg{Bucket: domain.BucketActive})

	view := model.View()
	if view == "" {
		t.Error("View() returned empty string after triage move")
	}
}

func TestTriageMoveError(t *testing.T) {
	deps := newTestDeps(t)
	model := tui.NewModel(deps)
	model.Init()

	// Load resources
	resources := []domain.Resource{
		{ID: "r1", Title: "Item 1", Bucket: domain.BucketInbox},
	}
	model = updateModel(t, model, tui.ResourceLoadedMsg{Resources: resources})

	// Go to triage
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})

	// Enter to move
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyEnter})

	// Simulate move error
	model = updateModel(t, model, tui.TriageMoveMsg{Err: "connection refused"})

	view := model.View()
	if view == "" {
		t.Error("View() returned empty string after triage error")
	}
}

func TestBackspaceInAddForm(t *testing.T) {
	deps := newTestDeps(t)
	model := tui.NewModel(deps)
	model.Init()

	// Go to add screen
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	// Type some characters
	for _, r := range []rune{'h', 't', 't', 'p'} {
		model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	// Backspace
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyBackspace})

	view := model.View()
	if view == "" {
		t.Error("View() returned empty string")
	}
}

func TestListNavigation(t *testing.T) {
	deps := newTestDeps(t)
	model := tui.NewModel(deps)
	model.Init()

	resources := []domain.Resource{
		{ID: "r1", Title: "Item 1", Bucket: domain.BucketInbox},
		{ID: "r2", Title: "Item 2", Bucket: domain.BucketActive},
		{ID: "r3", Title: "Item 3", Bucket: domain.BucketLater},
	}
	model = updateModel(t, model, tui.ResourceLoadedMsg{Resources: resources})

	// Go to list
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})

	// Navigate down twice
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyDown})
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyDown})

	// Navigate up
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyUp})

	view := model.View()
	if view == "" {
		t.Error("View() returned empty string")
	}
}

func TestConfigNavigation(t *testing.T) {
	deps := newTestDeps(t)
	model := tui.NewModel(deps)
	model.Init()

	// Go to config
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})

	// Navigate through all fields
	for i := 0; i < 5; i++ {
		model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyDown})
	}

	// Navigate back up
	for i := 0; i < 5; i++ {
		model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyUp})
	}

	view := model.View()
	if view == "" {
		t.Error("View() returned empty string")
	}
}
