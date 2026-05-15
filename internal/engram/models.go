package engram

// MCPResponse represents a parsed response from an Engram MCP tool call.
type MCPResponse struct {
	ID      string         `json:"id,omitempty"`
	Type    string         `json:"type,omitempty"`
	Content map[string]any `json:"content,omitempty"`
	Error   string         `json:"error,omitempty"`
}
