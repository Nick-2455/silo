package engram

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"

	"github.com/Nick-2455/silo/internal/domain"
)

const (
	defaultEngramBin = "engram"
	callTimeout      = 5 * time.Second
	connectTimeout   = 3 * time.Second
)

// MCPClient implements domain.EngramClient via MCP stdio to engram.
type MCPClient struct {
	client *client.Client
	mu     sync.Mutex
	closed bool
}

// NewClient creates a new MCP client that spawns "engram mcp --tools=agent".
// If engramPath is empty, "engram" is used (resolved from PATH).
func NewClient(engramPath string) (*MCPClient, error) {
	if engramPath == "" {
		engramPath = defaultEngramBin
	}

	c, err := client.NewStdioMCPClient(engramPath, nil, "mcp", "--tools=agent")
	if err != nil {
		return nil, fmt.Errorf("engram: create stdio client: %w", err)
	}

	// Initialize the MCP connection
	ctx, cancel := context.WithTimeout(context.Background(), connectTimeout)
	defer cancel()

	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{
		Name:    "silo",
		Version: "0.1.0",
	}

	if _, err := c.Initialize(ctx, initReq); err != nil {
		c.Close()
		return nil, fmt.Errorf("engram: initialize: %w", err)
	}

	return &MCPClient{client: c}, nil
}

// CreateResource creates a new resource in Engram via mem_save.
// Each resource is a unique observation — no topic_key is used to avoid upserts.
func (c *MCPClient) CreateResource(ctx context.Context, r domain.Resource) (string, error) {
	contentMap := map[string]string{
		"url":    r.URL,
		"title":  r.Title,
		"bucket": string(r.Bucket),
	}
	if r.Content != "" {
		contentMap["content"] = r.Content
	}
	contentBytes, err := json.Marshal(contentMap)
	if err != nil {
		return "", fmt.Errorf("engram: marshal content: %w", err)
	}

	result, err := c.callTool(ctx, "mem_save", map[string]any{
		"title":   r.Title,
		"content": string(contentBytes),
		"type":    "resource",
		"project": domain.DefaultProject,
	})
	if err != nil {
		return "", err
	}

	// Parse the observation ID from the response text
	text := extractText(result)
	return parseObservationID(text), nil
}

// GetResource retrieves a resource by ID via mem_get_observation.
func (c *MCPClient) GetResource(ctx context.Context, id string) (domain.Resource, error) {
	numID, err := parseIDToNumber(id)
	if err != nil {
		return domain.Resource{}, err
	}
	result, err := c.callTool(ctx, "mem_get_observation", map[string]any{
		"id": numID,
	})
	if err != nil {
		return domain.Resource{}, err
	}

	return parseResourceFromText(extractText(result))
}

// SearchResources searches for resources using mem_search.
func (c *MCPClient) SearchResources(ctx context.Context, query string) ([]domain.Resource, error) {
	result, err := c.callTool(ctx, "mem_search", map[string]any{
		"query":   query,
		"project": domain.DefaultProject,
		"type":    "resource",
		"limit":   50,
	})
	if err != nil {
		return nil, err
	}

	return parseSearchResults(extractText(result))
}

// SaveNode creates or updates a node in Engram with type and topic_key.
// Returns the Engram observation ID.
func (c *MCPClient) SaveNode(ctx context.Context, nodeType, title string, content map[string]any, topicKey, project string) (string, error) {
	contentBytes, err := json.Marshal(content)
	if err != nil {
		return "", fmt.Errorf("engram: marshal node content: %w", err)
	}

	result, err := c.callTool(ctx, "mem_save", map[string]any{
		"title":     title,
		"content":   string(contentBytes),
		"type":      nodeType,
		"topic_key": topicKey,
		"project":   project,
	})
	if err != nil {
		return "", err
	}

	text := extractText(result)
	return parseObservationID(text), nil
}

// UpdateNode updates a node's content by Engram observation ID.
func (c *MCPClient) UpdateNode(ctx context.Context, engramID string, content map[string]any) error {
	contentBytes, err := json.Marshal(content)
	if err != nil {
		return fmt.Errorf("engram: marshal update content: %w", err)
	}

	numID, err := parseIDToNumber(engramID)
	if err != nil {
		return err
	}

	_, err = c.callTool(ctx, "mem_update", map[string]any{
		"id":      numID,
		"content": string(contentBytes),
	})
	return err
}

// SearchNodes searches for nodes by query and optional type filter.
// Returns parsed observation IDs and titles from Engram search results.
func (c *MCPClient) SearchNodes(ctx context.Context, query, nodeType string) ([]domain.GraphNode, error) {
	args := map[string]any{
		"query":   query,
		"project": domain.DefaultProject,
		"limit":   50,
	}
	if nodeType != "" {
		args["type"] = nodeType
	}

	result, err := c.callTool(ctx, "mem_search", args)
	if err != nil {
		return nil, err
	}

	return parseGraphNodeResults(extractText(result))
}

// SearchByProject searches Engram for all observations under a given project.
// Used to discover pre-existing data for projects that may already exist in Engram.
func (c *MCPClient) SearchByProject(ctx context.Context, project string) ([]domain.DiscoveredObservation, error) {
	args := map[string]any{
		"query":   project,
		"project": project,
		"limit":   50,
	}

	result, err := c.callTool(ctx, "mem_search", args)
	if err != nil {
		return nil, err
	}

	return parseDiscoveredObservations(extractText(result))
}

// UpdateResource updates fields of an existing resource via mem_update.
func (c *MCPClient) UpdateResource(ctx context.Context, id string, updates map[string]any) error {
	// Build a content JSON string from the updates
	contentBytes, err := json.Marshal(updates)
	if err != nil {
		return fmt.Errorf("engram: marshal updates: %w", err)
	}

	numID, err := parseIDToNumber(id)
	if err != nil {
		return err
	}

	_, err = c.callTool(ctx, "mem_update", map[string]any{
		"id":      numID,
		"content": string(contentBytes),
	})
	return err
}

// IsReachable checks if the MCP client process is alive.
func (c *MCPClient) IsReachable(ctx context.Context) bool {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return false
	}
	c.mu.Unlock()

	// Try a lightweight tool call to verify connectivity
	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	_, err := c.callTool(pingCtx, "mem_search", map[string]any{
		"query": "",
		"limit": 1,
	})
	return err == nil
}

// Close shuts down the MCP client and its subprocess.
func (c *MCPClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true
	return c.client.Close()
}

// callTool executes an MCP tool call with a timeout.
func (c *MCPClient) callTool(ctx context.Context, name string, args map[string]any) (*mcp.CallToolResult, error) {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil, fmt.Errorf("engram: client closed")
	}
	c.mu.Unlock()

	callCtx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Name = name
	req.Params.Arguments = args

	result, err := c.client.CallTool(callCtx, req)
	if err != nil {
		return nil, fmt.Errorf("engram: call %s: %w", name, err)
	}

	if result.IsError {
		text := extractText(result)
		if text == "" {
			text = "unknown error"
		}
		return nil, fmt.Errorf("engram: tool %s error: %s", name, text)
	}

	return result, nil
}

// extractText pulls the first text content from a CallToolResult.
func extractText(result *mcp.CallToolResult) string {
	if result == nil || len(result.Content) == 0 {
		return ""
	}
	for _, c := range result.Content {
		if tc, ok := c.(mcp.TextContent); ok {
			return tc.Text
		}
	}
	return ""
}

// parseObservationID extracts an observation ID from mem_save response text.
// Engram mem_save returns a JSON object with an "id" field:
//
//	{"id":131,"project":"silo","result":"Memory saved: ..."}
//
// Falls back to extracting #<number> from plain text for older Engram versions.
func parseObservationID(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return text
	}

	// Primary path: parse JSON response and extract "id" field.
	var wrapper struct {
		ID json.Number `json:"id"`
	}
	if err := json.Unmarshal([]byte(text), &wrapper); err == nil && wrapper.ID.String() != "" {
		return wrapper.ID.String()
	}

	// Fallback: find "#<number>" pattern (older Engram versions).
	if idx := strings.Index(text, "#"); idx >= 0 {
		rest := text[idx+1:]
		// Take digits and stop at first non-digit
		end := 0
		for end < len(rest) && rest[end] >= '0' && rest[end] <= '9' {
			end++
		}
		if end > 0 {
			return rest[:end]
		}
	}
	return strings.TrimSpace(text)
}

// parseResourceFromText parses a single resource from mem_get_observation response text.
// The response is JSON: {"project":"silo","result":"#131 [resource] prueba1\n{...}\n..."}
func parseResourceFromText(text string) (domain.Resource, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return domain.Resource{}, fmt.Errorf("engram: empty response")
	}

	// Parse outer JSON wrapper
	var wrapper struct {
		Result string `json:"result"`
	}
	if err := json.Unmarshal([]byte(text), &wrapper); err == nil && wrapper.Result != "" {
		return parseMemoryBlock(wrapper.Result), nil
	}

	// Fallback: treat as raw memory block
	r := parseMemoryBlock(text)
	if r.ID == "" && r.Title == "" {
		return domain.Resource{}, fmt.Errorf("engram: could not parse resource from text")
	}
	return r, nil
}

// parseSearchResults parses mem_search text results into []domain.Resource.
// The response text is a JSON object: {"project":"silo","result":"Found N memories:\n\n..."}
// The result field contains memory blocks with ID, type, title, and content JSON.
func parseSearchResults(text string) ([]domain.Resource, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return []domain.Resource{}, nil
	}

	// Parse outer JSON wrapper
	var wrapper struct {
		Result string `json:"result"`
	}
	if err := json.Unmarshal([]byte(text), &wrapper); err != nil {
		return nil, fmt.Errorf("engram: parse search wrapper: %w", err)
	}

	if wrapper.Result == "" {
		return []domain.Resource{}, nil
	}

	return parseMemoryBlocks(wrapper.Result), nil
}

// parseMemoryBlocks extracts resource entries from Engram's search result text.
// Format: "[N] #ID (type) — title\n    {content JSON}\n    metadata..."
func parseMemoryBlocks(text string) []domain.Resource {
	var resources []domain.Resource

	// Each memory block is separated by a blank line
	blocks := strings.Split(text, "\n\n")
	for _, block := range blocks {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}

		r := parseMemoryBlock(block)
		if r.ID != "" || r.Title != "" {
			resources = append(resources, r)
		}
	}

	return resources
}

// parseMemoryBlock parses a single memory block into a domain.Resource.
func parseMemoryBlock(block string) domain.Resource {
	var r domain.Resource
	lines := strings.Split(block, "\n")

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Line 0: "[1] #131 (resource) — prueba1"
		if i == 0 {
			r = parseHeaderLine(line)
			continue
		}

		// Try to parse as JSON content
		if strings.HasPrefix(line, "{") {
			parseContentLine(line, &r)
			continue
		}
	}

	return r
}

// parseHeaderLine extracts ID, type, and title from "[1] #131 (resource) — prueba1"
func parseHeaderLine(line string) domain.Resource {
	var r domain.Resource

	// Extract "#ID" 
	if idx := strings.Index(line, "#"); idx >= 0 {
		rest := line[idx+1:]
		if space := strings.IndexAny(rest, " \t("); space > 0 {
			r.ID = rest[:space]
		}
	}

	// Extract title after "—"
	if idx := strings.Index(line, "—"); idx >= 0 {
		r.Title = strings.TrimSpace(line[idx+len("—"):])
	}

	return r
}

// parseContentLine parses the JSON content line and updates the resource.
func parseContentLine(line string, r *domain.Resource) {
	var content map[string]string
	if err := json.Unmarshal([]byte(line), &content); err != nil {
		return
	}

	if url, ok := content["url"]; ok {
		r.URL = url
	}
	if title, ok := content["title"]; ok {
		r.Title = title
	}
	if bucket, ok := content["bucket"]; ok {
		r.Bucket = domain.Bucket(bucket)
	}
}

// parseIDToNumber converts a string observation ID (e.g. "131") to a float64
// for use with Engram MCP tools that expect numeric IDs (mem_get_observation, mem_update).
func parseIDToNumber(id string) (float64, error) {
	n, err := strconv.ParseFloat(id, 64)
	if err != nil {
		return 0, fmt.Errorf("engram: invalid observation ID %q: %w", id, err)
	}
	return n, nil
}

// parseGraphNodeResults parses mem_search text results into []domain.GraphNode.
// The response text follows the same format as resource search results.
func parseGraphNodeResults(text string) ([]domain.GraphNode, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return []domain.GraphNode{}, nil
	}

	// Parse outer JSON wrapper
	var wrapper struct {
		Result string `json:"result"`
	}
	if err := json.Unmarshal([]byte(text), &wrapper); err != nil {
		return nil, fmt.Errorf("engram: parse graph search wrapper: %w", err)
	}

	if wrapper.Result == "" {
		return []domain.GraphNode{}, nil
	}

	// Parse memory blocks — each block has header line with #ID (type) — title
	var nodes []domain.GraphNode
	blocks := strings.Split(wrapper.Result, "\n\n")
	for _, block := range blocks {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}

		lines := strings.Split(block, "\n")
		if len(lines) == 0 {
			continue
		}

		// Parse header: "[1] #131 (domain) — Backend Development"
		header := lines[0]
		var node domain.GraphNode

		// Extract ID
		if idx := strings.Index(header, "#"); idx >= 0 {
			rest := header[idx+1:]
			if space := strings.IndexAny(rest, " \t("); space > 0 {
				node.EngramID = rest[:space]
			}
		}

		// Extract type
		if start := strings.Index(header, "("); start >= 0 {
			end := strings.Index(header[start:], ")")
			if end > 0 {
				nodeType := header[start+1 : start+end]
				node.NodeType = domain.NodeType(nodeType)
			}
		}

		// Extract title after "—"
		if idx := strings.Index(header, "—"); idx >= 0 {
			node.Title = strings.TrimSpace(header[idx+len("—"):])
		}

		node.Active = true // Default to active for search results

		if node.EngramID != "" || node.Title != "" {
			nodes = append(nodes, node)
		}
	}

	return nodes, nil
}

// parseDiscoveredObservations parses mem_search results into []DiscoveredObservation.
func parseDiscoveredObservations(text string) ([]domain.DiscoveredObservation, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return []domain.DiscoveredObservation{}, nil
	}

	var wrapper struct {
		Result string `json:"result"`
	}
	if err := json.Unmarshal([]byte(text), &wrapper); err != nil {
		return nil, fmt.Errorf("engram: parse discover wrapper: %w", err)
	}

	if wrapper.Result == "" {
		return []domain.DiscoveredObservation{}, nil
	}

	var obs []domain.DiscoveredObservation
	blocks := strings.Split(wrapper.Result, "\n\n")
	for _, block := range blocks {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}

		lines := strings.Split(block, "\n")
		if len(lines) == 0 {
			continue
		}

		var o domain.DiscoveredObservation
		header := lines[0]

		if idx := strings.Index(header, "#"); idx >= 0 {
			rest := header[idx+1:]
			if space := strings.IndexAny(rest, " \t("); space > 0 {
				o.ID = rest[:space]
			}
		}

		if start := strings.Index(header, "("); start >= 0 {
			end := strings.Index(header[start:], ")")
			if end > 0 {
				o.Type = header[start+1 : start+end]
			}
		}

		if idx := strings.Index(header, "\u2014"); idx >= 0 {
			o.Title = strings.TrimSpace(header[idx+len("\u2014"):])
		} else if idx := strings.Index(header, "--"); idx >= 0 {
			o.Title = strings.TrimSpace(header[idx+2:])
		}

		// Extract session_id and content preview from JSON body
		sid, preview := parseContentFields(block)
		o.SessionID = sid
		o.ContentPreview = preview
		o.ImportableAs = domain.EngramTypeMap[o.Type]

		if o.ID != "" || o.Title != "" {
			obs = append(obs, o)
		}
	}

	return obs, nil
}

// parseContentFields extracts session_id and content preview from JSON body lines.
func parseContentFields(block string) (sessionID, contentPreview string) {
	lines := strings.Split(block, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "{") {
			continue
		}
		var fields map[string]any
		if err := json.Unmarshal([]byte(line), &fields); err != nil {
			continue
		}
		if sid, ok := fields["session_id"].(string); ok && sid != "" {
			sessionID = sid
		}
		if content, ok := fields["content"].(string); ok && content != "" {
			if len(content) > 200 {
				contentPreview = content[:197] + "..."
			} else {
				contentPreview = content
			}
		}
		// If no content field, look for other text fields for preview
		if contentPreview == "" {
			for _, key := range []string{"What", "Why", "description", "title"} {
				if val, ok := fields[key].(string); ok && val != "" {
					if len(val) > 200 {
						contentPreview = val[:197] + "..."
					} else {
						contentPreview = val
					}
					break
				}
			}
		}
	}
	return sessionID, contentPreview
}
