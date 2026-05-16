package knowledge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

// MCPCaller invokes a single MCP tool and returns the raw call result.
// internal/engram.MCPClient satisfies this with its existing client.
type MCPCaller interface {
	CallTool(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error)
}

// EngramMCPReader implements EngramReader on top of an Engram MCP client.
type EngramMCPReader struct {
	Caller MCPCaller
}

// NewEngramMCPReader builds a reader from an existing MCP caller.
func NewEngramMCPReader(caller MCPCaller) *EngramMCPReader {
	return &EngramMCPReader{Caller: caller}
}

// ReadKnowledge invokes Engram `mem_search` and normalizes the response into items.
func (r *EngramMCPReader) ReadKnowledge(ctx context.Context, q EngramQuery) ([]KnowledgeItem, error) {
	if r == nil || r.Caller == nil {
		return nil, errors.New("knowledge: engram caller is not configured")
	}
	query := strings.TrimSpace(q.Query)
	if query == "" {
		return nil, errors.New("knowledge: query is required")
	}

	args := map[string]any{"query": query}
	if q.Project != "" {
		args["project"] = q.Project
	}
	if q.Type != "" {
		args["type"] = q.Type
	}
	if q.Limit > 0 {
		args["limit"] = q.Limit
	}

	req := mcp.CallToolRequest{}
	req.Params.Name = "mem_search"
	req.Params.Arguments = args

	result, err := r.Caller.CallTool(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("engram mem_search: %w", err)
	}
	if result != nil && result.IsError {
		return nil, fmt.Errorf("engram mem_search returned error: %s", extractResultText(result))
	}

	return parseEngramSearchText(extractResultText(result), q.Project), nil
}

func extractResultText(result *mcp.CallToolResult) string {
	if result == nil {
		return ""
	}
	for _, c := range result.Content {
		if tc, ok := c.(mcp.TextContent); ok {
			return tc.Text
		}
	}
	return ""
}

func parseEngramSearchText(text, projectHint string) []KnowledgeItem {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	body := text
	var wrapper struct {
		Result  string `json:"result"`
		Project string `json:"project"`
	}
	if err := json.Unmarshal([]byte(text), &wrapper); err == nil {
		if wrapper.Result != "" {
			body = wrapper.Result
		}
		if projectHint == "" {
			projectHint = wrapper.Project
		}
	}

	if strings.HasPrefix(body, "No memories found") {
		return nil
	}

	blocks := strings.Split(body, "\n\n")
	items := make([]KnowledgeItem, 0, len(blocks))
	for _, block := range blocks {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		item, ok := parseEngramBlock(block, projectHint)
		if !ok {
			continue
		}
		items = append(items, item)
	}
	return items
}

func parseEngramBlock(block, projectHint string) (KnowledgeItem, bool) {
	lines := strings.Split(block, "\n")
	if len(lines) == 0 {
		return KnowledgeItem{}, false
	}

	id, typ, title := parseHeader(lines[0])
	if id == "" && title == "" {
		return KnowledgeItem{}, false
	}

	preview := ""
	for _, line := range lines[1:] {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "[preview]") {
			continue
		}
		preview = trimmed
		if strings.HasSuffix(preview, "[preview]") {
			preview = strings.TrimSpace(strings.TrimSuffix(preview, "[preview]"))
		}
		break
	}

	item := KnowledgeItem{
		ID:      id,
		Title:   title,
		Type:    typ,
		Project: projectHint,
		Preview: preview,
		Source:  "engram",
	}
	return item, true
}

func parseHeader(line string) (id, typ, title string) {
	if idx := strings.Index(line, "#"); idx >= 0 {
		rest := line[idx+1:]
		end := 0
		for end < len(rest) && rest[end] >= '0' && rest[end] <= '9' {
			end++
		}
		if end > 0 {
			id = rest[:end]
		}
	}
	if start := strings.Index(line, "("); start >= 0 {
		if end := strings.Index(line[start:], ")"); end > 0 {
			typ = strings.TrimSpace(line[start+1 : start+end])
		}
	}
	if idx := strings.Index(line, "—"); idx >= 0 {
		title = strings.TrimSpace(line[idx+len("—"):])
	}
	return id, typ, title
}
