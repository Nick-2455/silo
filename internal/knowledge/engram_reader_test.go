package knowledge

import (
	"context"
	"errors"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

type stubCaller struct {
	last    mcp.CallToolRequest
	result  *mcp.CallToolResult
	err     error
}

func (s *stubCaller) CallTool(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.last = req
	if s.err != nil {
		return nil, s.err
	}
	return s.result, nil
}

func textResult(text string, isErr bool) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: text}},
		IsError: isErr,
	}
}

func TestEngramMCPReader_ParsesMemorySearchResponse(t *testing.T) {
	body := `{"project":"silo","result":"Found 2 memories:\n\n[1] #248 (architecture) — SDD proposal Silo Engram Obsidian MVP\n    Silo as bridge.... [preview]\n\n[2] #249 (architecture) — SDD spec Silo Engram Obsidian MVP\n    Specs detail. [preview]"}`
	caller := &stubCaller{result: textResult(body, false)}
	reader := NewEngramMCPReader(caller)

	items, err := reader.ReadKnowledge(context.Background(), EngramQuery{Query: "silo", Project: "silo", Type: "architecture", Limit: 5})
	if err != nil {
		t.Fatalf("read knowledge: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	got := items[0]
	if got.ID != "248" || got.Title != "SDD proposal Silo Engram Obsidian MVP" {
		t.Fatalf("unexpected first item header: %+v", got)
	}
	if got.Type != "architecture" || got.Project != "silo" || got.Source != "engram" {
		t.Fatalf("unexpected first item metadata: %+v", got)
	}
	if got.Preview != "Silo as bridge...." {
		t.Fatalf("unexpected preview: %q", got.Preview)
	}
	if caller.last.Params.Name != "mem_search" {
		t.Fatalf("expected mem_search call, got %q", caller.last.Params.Name)
	}
	args, ok := caller.last.Params.Arguments.(map[string]any)
	if !ok {
		t.Fatalf("expected map arguments, got %T", caller.last.Params.Arguments)
	}
	if args["project"] != "silo" || args["type"] != "architecture" || args["limit"].(int) != 5 {
		t.Fatalf("unexpected mem_search args: %+v", args)
	}
}

func TestEngramMCPReader_EmptyOnNoMemories(t *testing.T) {
	body := `{"project":"silo","result":"No memories found for: \"foo\""}`
	caller := &stubCaller{result: textResult(body, false)}
	reader := NewEngramMCPReader(caller)

	items, err := reader.ReadKnowledge(context.Background(), EngramQuery{Query: "foo"})
	if err != nil {
		t.Fatalf("read knowledge: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected no items, got %d", len(items))
	}
}

func TestEngramMCPReader_RequiresQuery(t *testing.T) {
	caller := &stubCaller{result: textResult("", false)}
	reader := NewEngramMCPReader(caller)
	if _, err := reader.ReadKnowledge(context.Background(), EngramQuery{}); err == nil {
		t.Fatalf("expected error on empty query")
	}
}

func TestEngramMCPReader_SurfacesCallErrors(t *testing.T) {
	caller := &stubCaller{err: errors.New("transport down")}
	reader := NewEngramMCPReader(caller)
	_, err := reader.ReadKnowledge(context.Background(), EngramQuery{Query: "x"})
	if err == nil {
		t.Fatalf("expected transport error")
	}
}

func TestEngramMCPReader_SurfacesEngramError(t *testing.T) {
	caller := &stubCaller{result: textResult("ambiguous_project", true)}
	reader := NewEngramMCPReader(caller)
	_, err := reader.ReadKnowledge(context.Background(), EngramQuery{Query: "x"})
	if err == nil {
		t.Fatalf("expected engram tool error")
	}
}
