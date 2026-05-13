package mcp

import (
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	serverName    = "marrow"
	serverVersion = "0.1.0"
)

// NewServer creates an MCP server with all tools registered.
func NewServer() *server.MCPServer {
	s := server.NewMCPServer(
		serverName,
		serverVersion,
		server.WithToolCapabilities(false),
	)

	// Register tools
	s.AddTool(searchTool(), handleSearch)
	s.AddTool(addResourceTool(), handleAddResource)
	s.AddTool(getRoadmapTool(), handleGetRoadmap)
	s.AddTool(triageTool(), handleTriage)

	return s
}

func searchTool() mcp.Tool {
	return mcp.NewTool("search",
		mcp.WithDescription("Search for resources in the knowledge graph. Queries Engram with cache fallback."),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Search query for FTS5 search"),
		),
	)
}

func addResourceTool() mcp.Tool {
	return mcp.NewTool("add_resource",
		mcp.WithDescription("Add a new resource to the knowledge graph. Creates an Engram observation in the inbox bucket."),
		mcp.WithString("url",
			mcp.Required(),
			mcp.Description("URL of the resource"),
		),
		mcp.WithString("title",
			mcp.Description("Title of the resource"),
		),
		mcp.WithString("content",
			mcp.Description("Content or notes about the resource"),
		),
	)
}

func getRoadmapTool() mcp.Tool {
	return mcp.NewTool("get_roadmap",
		mcp.WithDescription("Get the full roadmap with resources grouped by bucket (inbox, active, later, archived)."),
	)
}

func triageTool() mcp.Tool {
	return mcp.NewTool("triage",
		mcp.WithDescription("Move a resource to a different bucket. Updates both local state and Engram."),
		mcp.WithString("id",
			mcp.Required(),
			mcp.Description("Resource ID (Engram observation ID)"),
		),
		mcp.WithString("bucket",
			mcp.Required(),
			mcp.Description("Target bucket: inbox, active, later, or archived"),
			mcp.Enum("inbox", "active", "later", "archived"),
		),
	)
}
