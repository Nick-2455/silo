package mcp

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/Nick-2455/silo/internal/domain"
)

// mockEngramClient is a test double for domain.EngramClient.
type mockEngramClient struct {
	createResourceFn func(ctx context.Context, r domain.Resource) (string, error)
	searchFn         func(ctx context.Context, query string) ([]domain.Resource, error)
	getResourceFn    func(ctx context.Context, id string) (domain.Resource, error)
	updateFn         func(ctx context.Context, id string, updates map[string]any) error
	isReachableFn    func(ctx context.Context) bool
}

func (m *mockEngramClient) CreateResource(ctx context.Context, r domain.Resource) (string, error) {
	if m.createResourceFn != nil {
		return m.createResourceFn(ctx, r)
	}
	return "", nil
}

func (m *mockEngramClient) GetResource(ctx context.Context, id string) (domain.Resource, error) {
	if m.getResourceFn != nil {
		return m.getResourceFn(ctx, id)
	}
	return domain.Resource{}, nil
}

func (m *mockEngramClient) SearchResources(ctx context.Context, query string) ([]domain.Resource, error) {
	if m.searchFn != nil {
		return m.searchFn(ctx, query)
	}
	return nil, nil
}

func (m *mockEngramClient) UpdateResource(ctx context.Context, id string, updates map[string]any) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, id, updates)
	}
	return nil
}

func (m *mockEngramClient) IsReachable(ctx context.Context) bool {
	if m.isReachableFn != nil {
		return m.isReachableFn(ctx)
	}
	return true
}

func (m *mockEngramClient) SaveNode(ctx context.Context, nodeType, title string, content map[string]any, topicKey, project string) (string, error) {
	return "test-node-id", nil
}

func (m *mockEngramClient) UpdateNode(ctx context.Context, engramID string, content map[string]any) error {
	return nil
}

func (m *mockEngramClient) SearchByProject(ctx context.Context, project string) ([]domain.DiscoveredObservation, error) {
	return nil, nil
}

// mockStore is a test double for domain.ResourceStore.
type mockStore struct {
	getTriageFn    func(ctx context.Context, resourceID string) (domain.TriagePosition, error)
	setTriageFn    func(ctx context.Context, pos domain.TriagePosition) error
	getAllFn       func(ctx context.Context) ([]domain.TriagePosition, error)
	getCacheFn     func(ctx context.Context, query string) ([]domain.Resource, bool, error)
	cacheSearchFn  func(ctx context.Context, query string, results []domain.Resource) error
	invalidCacheFn func(ctx context.Context) error
}

func (m *mockStore) GetTriagePosition(ctx context.Context, resourceID string) (domain.TriagePosition, error) {
	if m.getTriageFn != nil {
		return m.getTriageFn(ctx, resourceID)
	}
	return domain.TriagePosition{}, domain.ErrTriageNotFound
}

func (m *mockStore) SetTriagePosition(ctx context.Context, pos domain.TriagePosition) error {
	if m.setTriageFn != nil {
		return m.setTriageFn(ctx, pos)
	}
	return nil
}

func (m *mockStore) GetAllTriagePositions(ctx context.Context) ([]domain.TriagePosition, error) {
	if m.getAllFn != nil {
		return m.getAllFn(ctx)
	}
	return nil, nil
}

func (m *mockStore) CacheSearch(ctx context.Context, query string, results []domain.Resource) error {
	if m.cacheSearchFn != nil {
		return m.cacheSearchFn(ctx, query, results)
	}
	return nil
}

func (m *mockStore) GetCachedSearch(ctx context.Context, query string) ([]domain.Resource, bool, error) {
	if m.getCacheFn != nil {
		return m.getCacheFn(ctx, query)
	}
	return nil, false, nil
}

func (m *mockStore) Close() error {
	return nil
}

func (m *mockStore) InvalidateSearchCache(ctx context.Context) error {
	if m.invalidCacheFn != nil {
		return m.invalidCacheFn(ctx)
	}
	return nil
}

// mockGraphStore is a test double for domain.GraphStore.
type mockGraphStore struct {
	getDomainTreeFn  func(ctx context.Context) ([]domain.DomainWithSubareas, error)
	listNodesByTypeFn func(ctx context.Context, nodeType domain.NodeType) ([]domain.GraphNode, error)
	getEdgesFn       func(ctx context.Context, nodeID, direction string) ([]domain.GraphEdge, error)
	getNodeFn        func(ctx context.Context, engramID string) (domain.GraphNode, error)
	upsertNodeFn     func(ctx context.Context, node domain.GraphNode) error
	addEdgeFn        func(ctx context.Context, fromID, toID string, label domain.EdgeLabel) error
	deleteNodeFn     func(ctx context.Context, engramID string) error
}

func (m *mockGraphStore) UpsertNode(ctx context.Context, node domain.GraphNode) error {
	if m.upsertNodeFn != nil {
		return m.upsertNodeFn(ctx, node)
	}
	return nil
}

func (m *mockGraphStore) DeleteNode(ctx context.Context, engramID string) error {
	if m.deleteNodeFn != nil {
		return m.deleteNodeFn(ctx, engramID)
	}
	return nil
}

func (m *mockGraphStore) GetNode(ctx context.Context, engramID string) (domain.GraphNode, error) {
	if m.getNodeFn != nil {
		return m.getNodeFn(ctx, engramID)
	}
	return domain.GraphNode{}, domain.ErrNodeNotFound
}

func (m *mockGraphStore) ListNodesByType(ctx context.Context, nodeType domain.NodeType) ([]domain.GraphNode, error) {
	if m.listNodesByTypeFn != nil {
		return m.listNodesByTypeFn(ctx, nodeType)
	}
	return nil, nil
}

func (m *mockGraphStore) AddEdge(ctx context.Context, fromID, toID string, label domain.EdgeLabel) error {
	if m.addEdgeFn != nil {
		return m.addEdgeFn(ctx, fromID, toID, label)
	}
	return nil
}

func (m *mockGraphStore) RemoveEdge(ctx context.Context, fromID, toID string, label domain.EdgeLabel) error {
	return nil
}

func (m *mockGraphStore) GetEdges(ctx context.Context, nodeID, direction string) ([]domain.GraphEdge, error) {
	if m.getEdgesFn != nil {
		return m.getEdgesFn(ctx, nodeID, direction)
	}
	return nil, nil
}

func (m *mockGraphStore) GetNeighbors(ctx context.Context, nodeID string, label domain.EdgeLabel) ([]domain.GraphNode, error) {
	return nil, nil
}

func (m *mockGraphStore) GetDomainTree(ctx context.Context) ([]domain.DomainWithSubareas, error) {
	if m.getDomainTreeFn != nil {
		return m.getDomainTreeFn(ctx)
	}
	return nil, nil
}

func (m *mockGraphStore) ListActiveProjects(ctx context.Context) ([]domain.Project, error) {
	return nil, nil
}

func (m *mockGraphStore) ListSessions(ctx context.Context, projectID string) ([]domain.Session, error) {
	return nil, nil
}

func (m *mockGraphStore) GetSession(ctx context.Context, id string) (domain.Session, error) {
	return domain.Session{}, domain.ErrSessionNotFound
}

func (m *mockGraphStore) ListLearnings(ctx context.Context, subareaID string) ([]domain.Learning, error) {
	return nil, nil
}

func (m *mockGraphStore) GetLearning(ctx context.Context, id string) (domain.Learning, error) {
	return domain.Learning{}, domain.ErrLearningNotFound
}

func (m *mockGraphStore) UpsertPerson(ctx context.Context, node domain.GraphNode) error {
	return nil
}

func (m *mockGraphStore) Close() error {
	return nil
}

func newTestDeps() *Deps {
	return &Deps{
		Engram:     &mockEngramClient{},
		Store:      &mockStore{},
		GraphStore: &mockGraphStore{},
	}
}

func TestHandleSearch_EngramSuccess(t *testing.T) {
	handlerDeps = &Deps{
		Engram: &mockEngramClient{
			searchFn: func(ctx context.Context, query string) ([]domain.Resource, error) {
				return []domain.Resource{
					{ID: "1", Title: "Test Resource", URL: "https://example.com", Bucket: domain.BucketInbox},
				}, nil
			},
		},
		Store:      &mockStore{},
		GraphStore: &mockGraphStore{},
	}

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"query": "test"}

	result, err := handleSearch(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &data); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	if data["degraded"] != false {
		t.Errorf("expected degraded=false, got %v", data["degraded"])
	}
	if int(data["count"].(float64)) != 1 {
		t.Errorf("expected count=1, got %v", data["count"])
	}
}

func TestHandleSearch_CacheFallback(t *testing.T) {
	cached := []domain.Resource{
		{ID: "cached-1", Title: "Cached Result", URL: "https://cached.com", Bucket: domain.BucketInbox},
	}

	handlerDeps = &Deps{
		Engram: &mockEngramClient{
			searchFn: func(ctx context.Context, query string) ([]domain.Resource, error) {
				return nil, domain.ErrEngramUnreachable
			},
		},
		Store: &mockStore{
			getCacheFn: func(ctx context.Context, query string) ([]domain.Resource, bool, error) {
				return cached, true, nil
			},
		},
		GraphStore: &mockGraphStore{},
	}

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"query": "test"}

	result, err := handleSearch(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &data); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	if data["degraded"] != true {
		t.Errorf("expected degraded=true, got %v", data["degraded"])
	}
}

func TestHandleAddResource_InvalidURL(t *testing.T) {
	handlerDeps = newTestDeps()

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"url": "not-a-url"}

	result, err := handleAddResource(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Content) == 0 {
		t.Fatal("expected error content")
	}

	text := result.Content[0].(mcp.TextContent).Text
	if text == "" {
		t.Error("expected error message for invalid URL")
	}
}

func TestHandleAddResource_Success(t *testing.T) {
	handlerDeps = &Deps{
		Engram: &mockEngramClient{
			createResourceFn: func(ctx context.Context, r domain.Resource) (string, error) {
				return "obs-123", nil
			},
		},
		Store:      &mockStore{},
		GraphStore: &mockGraphStore{},
	}

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"url":     "https://example.com/article",
		"title":   "Test Article",
		"content": "Some content",
	}

	result, err := handleAddResource(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &data); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	if data["id"] != "obs-123" {
		t.Errorf("expected id=obs-123, got %v", data["id"])
	}
	if data["bucket"] != "inbox" {
		t.Errorf("expected bucket=inbox, got %v", data["bucket"])
	}
}

func TestHandleGetRoadmap_Success(t *testing.T) {
	handlerDeps = &Deps{
		Engram: &mockEngramClient{
			getResourceFn: func(ctx context.Context, id string) (domain.Resource, error) {
				return domain.Resource{
					ID: id, Title: "New Resource", URL: "https://example.com", Bucket: domain.BucketInbox,
				}, nil
			},
		},
		Store: &mockStore{
			getAllFn: func(ctx context.Context) ([]domain.TriagePosition, error) {
				return []domain.TriagePosition{
					{ResourceID: "1", Bucket: domain.BucketInbox},
				}, nil
			},
		},
		GraphStore: &mockGraphStore{},
	}

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{}

	result, err := handleGetRoadmap(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &data); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	// Should have all 4 buckets
	if _, ok := data["inbox"]; !ok {
		t.Error("expected inbox bucket in roadmap")
	}
}

func TestHandleGetRoadmap_EngramDown(t *testing.T) {
	handlerDeps = &Deps{
		Engram: &mockEngramClient{
			getResourceFn: func(ctx context.Context, id string) (domain.Resource, error) {
				return domain.Resource{}, domain.ErrEngramUnreachable
			},
		},
		Store: &mockStore{
			getAllFn: func(ctx context.Context) ([]domain.TriagePosition, error) {
				return []domain.TriagePosition{
					{ResourceID: "1", Bucket: domain.BucketInbox},
				}, nil
			},
		},
		GraphStore: &mockGraphStore{},
	}

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{}

	result, err := handleGetRoadmap(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &data); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	if data["error"] == "" {
		t.Error("expected error message when Engram is down")
	}
}

func TestHandleTriage_InvalidBucket(t *testing.T) {
	handlerDeps = newTestDeps()

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"id":     "obs-123",
		"bucket": "invalid-bucket",
	}

	result, err := handleTriage(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	text := result.Content[0].(mcp.TextContent).Text
	if text == "" {
		t.Error("expected error message for invalid bucket")
	}
}

func TestHandleTriage_Success(t *testing.T) {
	handlerDeps = &Deps{
		Engram: &mockEngramClient{
			updateFn: func(ctx context.Context, id string, updates map[string]any) error {
				return nil
			},
		},
		Store:      &mockStore{},
		GraphStore: &mockGraphStore{},
	}

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"id":     "obs-123",
		"bucket": "active",
	}

	result, err := handleTriage(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &data); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	if data["bucket"] != "active" {
		t.Errorf("expected bucket=active, got %v", data["bucket"])
	}
}

func TestHandleTriage_RollbackOnEngramFailure(t *testing.T) {
	rollbackCalled := false

	handlerDeps = &Deps{
		Engram: &mockEngramClient{
			updateFn: func(ctx context.Context, id string, updates map[string]any) error {
				return domain.ErrEngramUnreachable
			},
		},
		Store: &mockStore{
			getTriageFn: func(ctx context.Context, resourceID string) (domain.TriagePosition, error) {
				return domain.TriagePosition{
					ResourceID: resourceID,
					Bucket:     domain.BucketInbox,
					UpdatedAt:  time.Now(),
				}, nil
			},
			setTriageFn: func(ctx context.Context, pos domain.TriagePosition) error {
				if pos.Bucket == domain.BucketInbox && pos.ResourceID == "obs-123" {
					rollbackCalled = true
				}
				return nil
			},
		},
		GraphStore: &mockGraphStore{},
	}

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"id":     "obs-123",
		"bucket": "active",
	}

	result, err := handleTriage(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	text := result.Content[0].(mcp.TextContent).Text
	if text == "" {
		t.Fatal("expected error message")
	}

	if !rollbackCalled {
		t.Error("expected rollback to be called on Engram failure")
	}
}

// TestProjectRouting verifies that each node type passes the correct Engram project.
func TestProjectRouting_CreateDomain(t *testing.T) {
	var gotProject string
	handlerDeps = &Deps{
		Engram: &trackingEngramClient{saveNodeFn: func(_ context.Context, _ string, _ string, _ map[string]any, _ string, project string) (string, error) {
			gotProject = project
			return "engram-1", nil
		}},
		GraphStore: &mockGraphStore{},
	}

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"name": "Dev"}
	_, _ = handleCreateDomain(context.Background(), req)

	if gotProject != domain.DefaultProject {
		t.Errorf("domain should use DefaultProject=%q, got %q", domain.DefaultProject, gotProject)
	}
}

func TestProjectRouting_CreateSubarea(t *testing.T) {
	var gotProject string
	handlerDeps = &Deps{
		Engram: &trackingEngramClient{saveNodeFn: func(_ context.Context, _ string, _ string, _ map[string]any, _ string, project string) (string, error) {
			gotProject = project
			return "engram-2", nil
		}},
		GraphStore: &mockGraphStore{
			listNodesByTypeFn: func(_ context.Context, nt domain.NodeType) ([]domain.GraphNode, error) {
				if nt == domain.NodeTypeDomain {
					return []domain.GraphNode{{EngramID: "domain/dev", Title: "Dev", Active: true}}, nil
				}
				return nil, nil
			},
			upsertNodeFn: func(_ context.Context, _ domain.GraphNode) error { return nil },
			addEdgeFn:    func(_ context.Context, _, _ string, _ domain.EdgeLabel) error { return nil },
		},
	}

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"name": "Backend", "domain_slug": "dev"}
	_, _ = handleCreateSubarea(context.Background(), req)

	if gotProject != domain.DefaultProject {
		t.Errorf("subarea should use DefaultProject=%q, got %q", domain.DefaultProject, gotProject)
	}
}

func TestProjectRouting_CreateProject(t *testing.T) {
	var gotProject string
	handlerDeps = &Deps{
		Engram: &trackingEngramClient{saveNodeFn: func(_ context.Context, _ string, _ string, _ map[string]any, _ string, project string) (string, error) {
			gotProject = project
			return "engram-3", nil
		}},
		GraphStore: &mockGraphStore{
			upsertNodeFn: func(_ context.Context, _ domain.GraphNode) error { return nil },
		},
	}

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"name": "cortex"}
	_, _ = handleCreateProject(context.Background(), req)

	if gotProject != "cortex" {
		t.Errorf("project should use its own slug='cortex', got %q", gotProject)
	}
}

func TestProjectRouting_CreateSession(t *testing.T) {
	var gotProject string
	handlerDeps = &Deps{
		Engram: &trackingEngramClient{saveNodeFn: func(_ context.Context, _ string, _ string, _ map[string]any, _ string, project string) (string, error) {
			gotProject = project
			return "engram-4", nil
		}},
		GraphStore: &mockGraphStore{
			listNodesByTypeFn: func(_ context.Context, nt domain.NodeType) ([]domain.GraphNode, error) {
				if nt == domain.NodeTypeProject {
					return []domain.GraphNode{{EngramID: "project/cortex", Title: "cortex", Active: true}}, nil
				}
				return nil, nil
			},
			upsertNodeFn: func(_ context.Context, _ domain.GraphNode) error { return nil },
			addEdgeFn:    func(_ context.Context, _, _ string, _ domain.EdgeLabel) error { return nil },
		},
	}

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"project_slug": "cortex", "description": "build API"}
	_, _ = handleCreateSession(context.Background(), req)

	if gotProject != "cortex" {
		t.Errorf("session should use parent project slug='cortex', got %q", gotProject)
	}
}

func TestProjectRouting_CreateLearning_ResolvesProjectFromSession(t *testing.T) {
	var gotProject string
	handlerDeps = &Deps{
		Engram: &trackingEngramClient{saveNodeFn: func(_ context.Context, _ string, _ string, _ map[string]any, _ string, project string) (string, error) {
			gotProject = project
			return "engram-5", nil
		}},
		GraphStore: &mockGraphStore{
			listNodesByTypeFn: func(_ context.Context, nt domain.NodeType) ([]domain.GraphNode, error) {
				if nt == domain.NodeTypeSession {
					return []domain.GraphNode{{EngramID: "session-1", Title: "build API", Active: true}}, nil
				}
				return nil, nil
			},
			getNodeFn: func(_ context.Context, id string) (domain.GraphNode, error) {
				if id == "project/cortex" {
					return domain.GraphNode{EngramID: "project/cortex", Title: "cortex", Active: true}, nil
				}
				return domain.GraphNode{}, domain.ErrNodeNotFound
			},
			getEdgesFn: func(_ context.Context, nodeID, direction string) ([]domain.GraphEdge, error) {
				if nodeID == "session-1" && direction == "to" {
					return []domain.GraphEdge{{FromID: "project/cortex", ToID: "session-1", Label: domain.EdgeWorkedOn}}, nil
				}
				return nil, nil
			},
			upsertNodeFn: func(_ context.Context, _ domain.GraphNode) error { return nil },
			addEdgeFn:    func(_ context.Context, _, _ string, _ domain.EdgeLabel) error { return nil },
		},
	}

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"session_slug": "build-api", "content": "learned something"}
	_, _ = handleCreateLearning(context.Background(), req)

	if gotProject != "cortex" {
		t.Errorf("learning should resolve parent project='cortex' from session edge, got %q", gotProject)
	}
}

func TestProjectRouting_CreateLearning_FallsBackToDefault(t *testing.T) {
	var gotProject string
	handlerDeps = &Deps{
		Engram: &trackingEngramClient{saveNodeFn: func(_ context.Context, _ string, _ string, _ map[string]any, _ string, project string) (string, error) {
			gotProject = project
			return "engram-6", nil
		}},
		GraphStore: &mockGraphStore{
			listNodesByTypeFn: func(_ context.Context, nt domain.NodeType) ([]domain.GraphNode, error) {
				if nt == domain.NodeTypeSession {
					return []domain.GraphNode{{EngramID: "session-1", Title: "orphan session", Active: true}}, nil
				}
				return nil, nil
			},
			getEdgesFn:    func(_ context.Context, _, _ string) ([]domain.GraphEdge, error) { return nil, nil },
			upsertNodeFn: func(_ context.Context, _ domain.GraphNode) error { return nil },
			addEdgeFn:    func(_ context.Context, _, _ string, _ domain.EdgeLabel) error { return nil },
		},
	}

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"session_slug": "orphan-session", "content": "orphan learning"}
	_, _ = handleCreateLearning(context.Background(), req)

	if gotProject != domain.DefaultProject {
		t.Errorf("learning without parent project should fall back to DefaultProject=%q, got %q", domain.DefaultProject, gotProject)
	}
}

// trackingEngramClient captures SaveNode calls for project routing assertions.
type trackingEngramClient struct {
	createResourceFn func(ctx context.Context, r domain.Resource) (string, error)
	saveNodeFn       func(ctx context.Context, nodeType, title string, content map[string]any, topicKey, project string) (string, error)
}

func (t *trackingEngramClient) CreateResource(ctx context.Context, r domain.Resource) (string, error) {
	if t.createResourceFn != nil {
		return t.createResourceFn(ctx, r)
	}
	return "", nil
}
func (t *trackingEngramClient) GetResource(_ context.Context, _ string) (domain.Resource, error) {
	return domain.Resource{}, nil
}
func (t *trackingEngramClient) SearchResources(_ context.Context, _ string) ([]domain.Resource, error) {
	return nil, nil
}
func (t *trackingEngramClient) UpdateResource(_ context.Context, _ string, _ map[string]any) error {
	return nil
}
func (t *trackingEngramClient) IsReachable(_ context.Context) bool { return true }
func (t *trackingEngramClient) SaveNode(ctx context.Context, nodeType, title string, content map[string]any, topicKey, project string) (string, error) {
	if t.saveNodeFn != nil {
		return t.saveNodeFn(ctx, nodeType, title, content, topicKey, project)
	}
	return "test-node-id", nil
}
func (t *trackingEngramClient) UpdateNode(_ context.Context, _ string, _ map[string]any) error { return nil }
func (t *trackingEngramClient) SearchByProject(_ context.Context, _ string) ([]domain.DiscoveredObservation, error) {
	return nil, nil
}
func (t *trackingEngramClient) Close() error { return nil }

// TestDiscoverProject verifies discover_project returns Engram observations by project.
func TestDiscoverProject(t *testing.T) {
	handlerDeps = &Deps{
		Engram: &searchByProjectMock{
			observations: []domain.DiscoveredObservation{
				{ID: "1", Type: "architecture", Title: "Cortex architecture", ImportableAs: "learning"},
				{ID: "2", Type: "decision", Title: "Use Bubble Tea", ImportableAs: "learning"},
				{ID: "3", Type: "bugfix", Title: "Fix search parsing", ImportableAs: "learning"},
				{ID: "4", Type: "architecture", Title: "MCP client design", ImportableAs: "learning"},
				{ID: "5", Type: "session_summary", Title: "Build session", ImportableAs: "session"},
				{ID: "6", Type: "tool_use", Title: "Ran go test", ImportableAs: ""},
			},
		},
		GraphStore: &mockGraphStore{},
	}

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"name": "cortex"}

	result, err := handleDiscoverProject(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &data); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	if data["project"] != "cortex" {
		t.Errorf("expected project=cortex, got %v", data["project"])
	}

	totalCount, ok := data["total_found"].(float64)
	if !ok || int(totalCount) != 6 {
		t.Errorf("expected total_found=6, got %v", data["total_found"])
	}

	importableCount, ok := data["importable"].(float64)
	if !ok || int(importableCount) != 5 {
		t.Errorf("expected importable=5, got %v", data["importable"])
	}

	skippedCount, ok := data["skipped"].(float64)
	if !ok || int(skippedCount) != 1 {
		t.Errorf("expected skipped=1, got %v", data["skipped"])
	}
}

func TestImportProject_ProjectNotFound(t *testing.T) {
	handlerDeps = &Deps{
		Engram:      &searchByProjectMock{},
		GraphStore:  &mockGraphStore{},
	}

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"name": "nonexistent"}

	result, err := handleImportProject(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	text := result.Content[0].(mcp.TextContent).Text
	if text == "" {
		t.Fatal("expected error message for missing project")
	}
}

func TestImportProject_MarksAlreadyImported(t *testing.T) {
	handlerDeps = &Deps{
		Engram: &searchByProjectMock{
			observations: []domain.DiscoveredObservation{
				{ID: "1", Type: "architecture", Title: "Already imported", ImportableAs: "learning"},
				{ID: "2", Type: "decision", Title: "New decision", ImportableAs: "learning"},
			},
		},
		GraphStore: &mockGraphStore{
			listNodesByTypeFn: func(_ context.Context, nt domain.NodeType) ([]domain.GraphNode, error) {
				if nt == domain.NodeTypeProject {
					return []domain.GraphNode{
						{EngramID: "project/cortex", Title: "cortex", Active: true},
					}, nil
				}
				return nil, nil
			},
			getNodeFn: func(_ context.Context, id string) (domain.GraphNode, error) {
				if id == "1" {
					return domain.GraphNode{EngramID: "1", Title: "Already imported", NodeType: domain.NodeTypeLearning}, nil
				}
				return domain.GraphNode{}, domain.ErrNodeNotFound
			},
			upsertNodeFn: func(_ context.Context, _ domain.GraphNode) error { return nil },
			addEdgeFn:    func(_ context.Context, _, _ string, _ domain.EdgeLabel) error { return nil },
		},
	}

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"name": "cortex"}

	result, err := handleImportProject(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &data); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	if data["project"] != "cortex" {
		t.Errorf("expected project=cortex, got %v", data["project"])
	}

	alreadyCount, ok := data["already_imported"].(float64)
	if !ok || int(alreadyCount) != 1 {
		t.Errorf("expected already_imported=1, got %v", data["already_imported"])
	}

	importedLearnings, ok := data["imported_learnings"].(float64)
	if !ok || int(importedLearnings) != 1 {
		t.Errorf("expected imported_learnings=1, got %v", data["imported_learnings"])
	}
}

func TestImportProject_AutoImportSession(t *testing.T) {
	handlerDeps = &Deps{
		Engram: &searchByProjectMock{
			observations: []domain.DiscoveredObservation{
				{ID: "77", Type: "session_summary", Title: "Session summary: blacksight", ImportableAs: "session"},
			},
			saveNodeFn: func(_ context.Context, _, _ string, _ map[string]any, _, _ string) (string, error) {
				return "77", nil
			},
		},
		GraphStore: &mockGraphStore{
			listNodesByTypeFn: func(_ context.Context, nt domain.NodeType) ([]domain.GraphNode, error) {
				if nt == domain.NodeTypeProject {
					return []domain.GraphNode{
						{EngramID: "project/blacksight", Title: "blacksight", Active: true},
					}, nil
				}
				return nil, nil
			},
			getNodeFn:    func(_ context.Context, _ string) (domain.GraphNode, error) { return domain.GraphNode{}, domain.ErrNodeNotFound },
			upsertNodeFn: func(_ context.Context, _ domain.GraphNode) error { return nil },
			addEdgeFn:    func(_ context.Context, _, _ string, _ domain.EdgeLabel) error { return nil },
		},
	}

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"name": "blacksight"}

	result, err := handleImportProject(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &data); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	if data["project"] != "blacksight" {
		t.Errorf("expected project=blacksight, got %v", data["project"])
	}

	importedSessions, ok := data["imported_sessions"].(float64)
	if !ok || int(importedSessions) != 1 {
		t.Errorf("expected imported_sessions=1, got %v", data["imported_sessions"])
	}
}

func TestImportProject_ReusesEngramID(t *testing.T) {
	saveNodeCalled := false
	handlerDeps = &Deps{
		Engram: &searchByProjectMock{
			observations: []domain.DiscoveredObservation{
				{ID: "68", Type: "architecture", Title: "SDD init: blacksight", ImportableAs: "learning"},
				{ID: "77", Type: "session_summary", Title: "Session summary: blacksight", ImportableAs: "session"},
				{ID: "99", Type: "discovery", Title: "New discovery", ImportableAs: "learning"},
			},
			saveNodeFn: func(_ context.Context, nodeType, title string, _ map[string]any, _, _ string) (string, error) {
				saveNodeCalled = true
				t.Errorf("SaveNode should NOT be called when observation already has an Engram ID, but was called for %q", title)
				return "unexpected", nil
			},
		},
		GraphStore: &mockGraphStore{
			listNodesByTypeFn: func(_ context.Context, nt domain.NodeType) ([]domain.GraphNode, error) {
				if nt == domain.NodeTypeProject {
					return []domain.GraphNode{
						{EngramID: "project/blacksight", Title: "blacksight", Active: true},
					}, nil
				}
				return nil, nil
			},
			getNodeFn:    func(_ context.Context, _ string) (domain.GraphNode, error) { return domain.GraphNode{}, domain.ErrNodeNotFound },
			upsertNodeFn: func(_ context.Context, _ domain.GraphNode) error { return nil },
			addEdgeFn:    func(_ context.Context, _, _ string, _ domain.EdgeLabel) error { return nil },
		},
	}

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"name": "blacksight"}

	result, err := handleImportProject(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if saveNodeCalled {
		t.Error("SaveNode should not have been called for observations with existing Engram IDs")
	}

	var data map[string]any
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &data); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	importedSessions, ok := data["imported_sessions"].(float64)
	if !ok || int(importedSessions) != 1 {
		t.Errorf("expected imported_sessions=1, got %v", data["imported_sessions"])
	}
	importedLearnings, ok := data["imported_learnings"].(float64)
	if !ok || int(importedLearnings) != 2 {
		t.Errorf("expected imported_learnings=2, got %v", data["imported_learnings"])
	}
	errors, ok := data["errors"].(float64)
	if !ok || int(errors) != 0 {
		t.Errorf("expected errors=0, got %v", data["errors"])
	}
}

type searchByProjectMock struct {
	observations []domain.DiscoveredObservation
	err          error
	saveNodeFn   func(ctx context.Context, nodeType, title string, content map[string]any, topicKey, project string) (string, error)
}

func (s *searchByProjectMock) CreateResource(_ context.Context, _ domain.Resource) (string, error) {
	return "1", nil
}
func (s *searchByProjectMock) GetResource(_ context.Context, _ string) (domain.Resource, error) {
	return domain.Resource{}, nil
}
func (s *searchByProjectMock) SearchResources(_ context.Context, _ string) ([]domain.Resource, error) {
	return nil, nil
}
func (s *searchByProjectMock) UpdateResource(_ context.Context, _ string, _ map[string]any) error { return nil }
func (s *searchByProjectMock) IsReachable(_ context.Context) bool                                   { return true }
func (s *searchByProjectMock) SaveNode(ctx context.Context, nodeType, title string, content map[string]any, topicKey, project string) (string, error) {
	if s.saveNodeFn != nil {
		return s.saveNodeFn(ctx, nodeType, title, content, topicKey, project)
	}
	return "1", nil
}
func (s *searchByProjectMock) UpdateNode(_ context.Context, _ string, _ map[string]any) error { return nil }
func (s *searchByProjectMock) SearchByProject(_ context.Context, project string) ([]domain.DiscoveredObservation, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.observations, nil
}
func (s *searchByProjectMock) Close() error { return nil }
