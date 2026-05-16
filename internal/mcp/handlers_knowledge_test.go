package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/Nick-2455/silo/internal/domain"
	"github.com/Nick-2455/silo/internal/knowledge"
)

type fakeKnowledgeService struct {
	readFn    func(ctx context.Context, q knowledge.EngramQuery) ([]knowledge.KnowledgeItem, error)
	syncFn    func(ctx context.Context, req knowledge.SyncRequest) (*knowledge.SyncReport, error)
	searchFn  func(ctx context.Context, vaultPath, query string, limit int) ([]knowledge.NoteSearchResult, error)
	writeFn   func(ctx context.Context, vaultPath string, note knowledge.Note) (knowledge.NoteWriteResult, error)
	contextFn func(ctx context.Context, req knowledge.ContextRequest) (*knowledge.KnowledgeContext, error)
}

func (f *fakeKnowledgeService) ReadKnowledge(ctx context.Context, q knowledge.EngramQuery) ([]knowledge.KnowledgeItem, error) {
	if f.readFn != nil {
		return f.readFn(ctx, q)
	}
	return nil, nil
}

func (f *fakeKnowledgeService) SyncToObsidian(ctx context.Context, req knowledge.SyncRequest) (*knowledge.SyncReport, error) {
	if f.syncFn != nil {
		return f.syncFn(ctx, req)
	}
	return &knowledge.SyncReport{}, nil
}

func (f *fakeKnowledgeService) SearchVault(ctx context.Context, vaultPath, query string, limit int) ([]knowledge.NoteSearchResult, error) {
	if f.searchFn != nil {
		return f.searchFn(ctx, vaultPath, query, limit)
	}
	return nil, nil
}

func (f *fakeKnowledgeService) CreateOrUpdateNote(ctx context.Context, vaultPath string, note knowledge.Note) (knowledge.NoteWriteResult, error) {
	if f.writeFn != nil {
		return f.writeFn(ctx, vaultPath, note)
	}
	return knowledge.NoteWriteResult{}, nil
}

func (f *fakeKnowledgeService) GetKnowledgeContext(ctx context.Context, req knowledge.ContextRequest) (*knowledge.KnowledgeContext, error) {
	if f.contextFn != nil {
		return f.contextFn(ctx, req)
	}
	return &knowledge.KnowledgeContext{Query: req.Query}, nil
}

func depsWithKnowledge(service KnowledgeService, vaultPath string) *Deps {
	cfg := domain.Config{ObsidianVaultPath: vaultPath}
	return &Deps{
		Engram:     &mockEngramClient{},
		Store:      &mockStore{},
		GraphStore: &mockGraphStore{},
		Knowledge:  service,
		Config:     cfg,
	}
}

func TestKnowledgeTools_DeclareStringSchemas(t *testing.T) {
	cases := map[string][]string{
		readFromEngramTool().Name:      {"query"},
		syncToObsidianTool().Name:      {"query"},
		searchVaultTool().Name:         {"query"},
		createOrUpdateNoteTool().Name:  {"title", "content"},
		getKnowledgeContextTool().Name: {"query"},
	}
	tools := []mcp.Tool{
		readFromEngramTool(),
		syncToObsidianTool(),
		searchVaultTool(),
		createOrUpdateNoteTool(),
		getKnowledgeContextTool(),
	}
	for _, tool := range tools {
		fields := cases[tool.Name]
		t.Run(tool.Name, func(t *testing.T) {
			for _, field := range fields {
				prop, ok := tool.InputSchema.Properties[field].(map[string]any)
				if !ok {
					t.Fatalf("tool %s missing schema for %s", tool.Name, field)
				}
				if prop["type"] != "string" {
					t.Fatalf("tool %s field %s type=%v", tool.Name, field, prop["type"])
				}
			}
		})
	}
}

func TestHandleReadFromEngram_PassesFiltersAndReturnsItems(t *testing.T) {
	fake := &fakeKnowledgeService{
		readFn: func(_ context.Context, q knowledge.EngramQuery) ([]knowledge.KnowledgeItem, error) {
			if q.Query != "silo" || q.Project != "silo" || q.Type != "architecture" || q.Limit != 5 {
				t.Fatalf("unexpected query: %+v", q)
			}
			return []knowledge.KnowledgeItem{{ID: "1", Title: "Engram", Source: "engram"}}, nil
		},
	}
	handlerDeps = depsWithKnowledge(fake, "")

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"query":   "silo",
		"project": "silo",
		"type":    "architecture",
		"limit":   float64(5),
	}
	res, err := handleReadFromEngram(context.Background(), req)
	if err != nil {
		t.Fatalf("handle read: %v", err)
	}
	payload := decodePayload(t, res)
	if payload["count"].(float64) != 1 {
		t.Fatalf("expected count=1, got %v", payload["count"])
	}
}

func TestHandleSyncToObsidian_UsesConfigVaultPath(t *testing.T) {
	vault := t.TempDir()
	fake := &fakeKnowledgeService{
		syncFn: func(_ context.Context, req knowledge.SyncRequest) (*knowledge.SyncReport, error) {
			if req.VaultPath != vault {
				t.Fatalf("expected vault from config, got %s", req.VaultPath)
			}
			return &knowledge.SyncReport{Total: 1, Written: 1}, nil
		},
	}
	handlerDeps = depsWithKnowledge(fake, vault)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"query": "memory"}
	res, err := handleSyncToObsidian(context.Background(), req)
	if err != nil {
		t.Fatalf("handle sync: %v", err)
	}
	payload := decodePayload(t, res)
	if payload["written"].(float64) != 1 {
		t.Fatalf("expected written=1, got %v", payload["written"])
	}
}

func TestHandleSyncToObsidian_VaultPathRequired(t *testing.T) {
	handlerDeps = depsWithKnowledge(&fakeKnowledgeService{}, "")
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"query": "memory"}
	res, err := handleSyncToObsidian(context.Background(), req)
	if err != nil {
		t.Fatalf("handle sync: %v", err)
	}
	if !strings.Contains(res.Content[0].(mcp.TextContent).Text, "vault path not configured") {
		t.Fatalf("expected vault error, got %s", res.Content[0].(mcp.TextContent).Text)
	}
}

func TestHandleSearchVault_PassesLimit(t *testing.T) {
	vault := t.TempDir()
	fake := &fakeKnowledgeService{
		searchFn: func(_ context.Context, gotVault, query string, limit int) ([]knowledge.NoteSearchResult, error) {
			if gotVault != vault || query != "memory" || limit != 5 {
				t.Fatalf("unexpected args: %s %s %d", gotVault, query, limit)
			}
			return []knowledge.NoteSearchResult{{Title: "Engram", Path: filepath.Join(vault, "Engram.md"), Snippet: "Engram memory"}}, nil
		},
	}
	handlerDeps = depsWithKnowledge(fake, vault)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"query": "memory",
		"limit": float64(5),
	}
	res, err := handleSearchVault(context.Background(), req)
	if err != nil {
		t.Fatalf("handle search: %v", err)
	}
	payload := decodePayload(t, res)
	if payload["count"].(float64) != 1 {
		t.Fatalf("unexpected payload: %v", payload)
	}
}

func TestHandleCreateOrUpdateNote_WritesAndReportsResult(t *testing.T) {
	vault := t.TempDir()
	fake := &fakeKnowledgeService{
		writeFn: func(_ context.Context, vaultPath string, note knowledge.Note) (knowledge.NoteWriteResult, error) {
			if vaultPath != vault || note.Title != "Bridge" || note.Content != "Body" {
				t.Fatalf("unexpected note: %+v in %s", note, vaultPath)
			}
			return knowledge.NoteWriteResult{Path: filepath.Join(vault, "Bridge.md"), Created: true}, nil
		},
	}
	handlerDeps = depsWithKnowledge(fake, vault)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"title":   "Bridge",
		"content": "Body",
	}
	res, err := handleCreateOrUpdateNote(context.Background(), req)
	if err != nil {
		t.Fatalf("handle create: %v", err)
	}
	payload := decodePayload(t, res)
	if payload["created"] != true {
		t.Fatalf("expected created=true, got %v", payload["created"])
	}
	if _, err := os.Stat(vault); err != nil {
		t.Fatalf("vault should exist: %v", err)
	}
}

func TestHandleGetKnowledgeContext_FailsOnEmptyQuery(t *testing.T) {
	handlerDeps = depsWithKnowledge(&fakeKnowledgeService{}, "")
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{}
	res, err := handleGetKnowledgeContext(context.Background(), req)
	if err != nil {
		t.Fatalf("handle context: %v", err)
	}
	if !strings.Contains(res.Content[0].(mcp.TextContent).Text, "query") {
		t.Fatalf("expected query error, got %s", res.Content[0].(mcp.TextContent).Text)
	}
}

func TestHandleGetKnowledgeContext_SurfacesEngramError(t *testing.T) {
	fake := &fakeKnowledgeService{
		contextFn: func(_ context.Context, _ knowledge.ContextRequest) (*knowledge.KnowledgeContext, error) {
			return nil, errors.New("engram offline")
		},
	}
	handlerDeps = depsWithKnowledge(fake, "")
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"query": "memory"}
	res, err := handleGetKnowledgeContext(context.Background(), req)
	if err != nil {
		t.Fatalf("handle context: %v", err)
	}
	if !strings.Contains(res.Content[0].(mcp.TextContent).Text, "engram offline") {
		t.Fatalf("expected engram offline error, got %s", res.Content[0].(mcp.TextContent).Text)
	}
}

func decodePayload(t *testing.T, res *mcp.CallToolResult) map[string]any {
	t.Helper()
	if len(res.Content) == 0 {
		t.Fatal("expected content")
	}
	text, ok := res.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected text content, got %T", res.Content[0])
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(text.Text), &payload); err != nil {
		t.Fatalf("decode payload: %v\n%s", err, text.Text)
	}
	return payload
}
