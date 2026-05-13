package mcp

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/Nick-2455/marrow/internal/domain"
)

// mockEngramClient is a test double for domain.EngramClient.
type mockEngramClient struct {
	createResourceFn func(ctx context.Context, r domain.Resource) (string, error)
	searchFn         func(ctx context.Context, query string) ([]domain.Resource, error)
	roadmapFn        func(ctx context.Context) (map[domain.Bucket][]domain.Resource, error)
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
	return domain.Resource{}, nil
}

func (m *mockEngramClient) SearchResources(ctx context.Context, query string) ([]domain.Resource, error) {
	if m.searchFn != nil {
		return m.searchFn(ctx, query)
	}
	return nil, nil
}

func (m *mockEngramClient) GetRoadmap(ctx context.Context) (map[domain.Bucket][]domain.Resource, error) {
	if m.roadmapFn != nil {
		return m.roadmapFn(ctx)
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

// mockStore is a test double for domain.ResourceStore.
type mockStore struct {
	getTriageFn    func(ctx context.Context, resourceID string) (domain.TriagePosition, error)
	setTriageFn    func(ctx context.Context, pos domain.TriagePosition) error
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

func newTestDeps() *Deps {
	return &Deps{
		Engram: &mockEngramClient{},
		Store:  &mockStore{},
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
		Store: &mockStore{},
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
		Store: &mockStore{},
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
	roadmap := map[domain.Bucket][]domain.Resource{
		domain.BucketInbox: {
			{ID: "1", Title: "New Resource", Bucket: domain.BucketInbox},
		},
		domain.BucketActive:  {},
		domain.BucketLater:   {},
		domain.BucketArchived: {},
	}

	handlerDeps = &Deps{
		Engram: &mockEngramClient{
			roadmapFn: func(ctx context.Context) (map[domain.Bucket][]domain.Resource, error) {
				return roadmap, nil
			},
		},
		Store: &mockStore{},
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
			roadmapFn: func(ctx context.Context) (map[domain.Bucket][]domain.Resource, error) {
				return nil, domain.ErrEngramUnreachable
			},
		},
		Store: &mockStore{},
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
		Store: &mockStore{},
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
