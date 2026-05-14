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

	// Resource tools
	s.AddTool(searchTool(), handleSearch)
	s.AddTool(addResourceTool(), handleAddResource)
	s.AddTool(getRoadmapTool(), handleGetRoadmap)
	s.AddTool(triageTool(), handleTriage)

	// Graph tools
	s.AddTool(listDomainsTool(), handleListDomains)
	s.AddTool(listProjectsTool(), handleListProjects)
	s.AddTool(createDomainTool(), handleCreateDomain)
	s.AddTool(createSubareaTool(), handleCreateSubarea)
	s.AddTool(createProjectTool(), handleCreateProject)
	s.AddTool(linkProjectTool(), handleLinkProject)
	s.AddTool(toggleProjectTool(), handleToggleProject)

	// Session and Learning tools
	s.AddTool(createSessionTool(), handleCreateSession)
	s.AddTool(createLearningTool(), handleCreateLearning)
	s.AddTool(listSessionsTool(), handleListSessions)
	s.AddTool(listLearningsTool(), handleListLearnings)
	s.AddTool(linkResourceTool(), handleLinkResource)
	s.AddTool(listPersonTool(), handleListPerson)
	s.AddTool(syncObsidianTool(), handleSyncObsidian)

	return s
}

// --- Resource tools ---

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

// --- Graph tools ---

func listDomainsTool() mcp.Tool {
	return mcp.NewTool("list_domains",
		mcp.WithDescription("List all domains with their nested subareas. Returns a hierarchical view of the knowledge taxonomy."),
	)
}

func listProjectsTool() mcp.Tool {
	return mcp.NewTool("list_projects",
		mcp.WithDescription("List all projects (active and inactive) with their linked subareas."),
	)
}

func createDomainTool() mcp.Tool {
	return mcp.NewTool("create_domain",
		mcp.WithDescription("Create a new knowledge domain. Stores in Engram and caches in SQLite."),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the domain"),
		),
		mcp.WithString("description",
			mcp.Description("Optional description of the domain"),
		),
	)
}

func createSubareaTool() mcp.Tool {
	return mcp.NewTool("create_subarea",
		mcp.WithDescription("Create a subarea under an existing domain. Adds a 'contains' edge from domain to subarea."),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the subarea"),
		),
		mcp.WithString("domain_slug",
			mcp.Required(),
			mcp.Description("Slug of the parent domain"),
		),
	)
}

func createProjectTool() mcp.Tool {
	return mcp.NewTool("create_project",
		mcp.WithDescription("Create a new project. Stores in Engram and caches in SQLite. Use link_project to connect to subareas."),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the project"),
		),
		mcp.WithString("description",
			mcp.Description("Optional description of the project"),
		),
	)
}

func linkProjectTool() mcp.Tool {
	return mcp.NewTool("link_project",
		mcp.WithDescription("Link a project to a subarea. Creates an 'applies_to' edge from project to subarea."),
		mcp.WithString("project_slug",
			mcp.Required(),
			mcp.Description("Slug of the project"),
		),
		mcp.WithString("subarea_slug",
			mcp.Required(),
			mcp.Description("Slug of the subarea"),
		),
	)
}

func toggleProjectTool() mcp.Tool {
	return mcp.NewTool("toggle_project",
		mcp.WithDescription("Toggle a project between active and inactive states. Updates both Engram and SQLite."),
		mcp.WithString("project_slug",
			mcp.Required(),
			mcp.Description("Slug of the project to toggle"),
		),
	)
}

// --- Session and Learning tools ---

func createSessionTool() mcp.Tool {
	return mcp.NewTool("create_session",
		mcp.WithDescription("Create a new session linked to a project. Creates a session node and a 'worked_on' edge from project to session."),
		mcp.WithString("project_slug",
			mcp.Required(),
			mcp.Description("Slug of the project this session belongs to"),
		),
		mcp.WithString("description",
			mcp.Required(),
			mcp.Description("Description of the session"),
		),
	)
}

func createLearningTool() mcp.Tool {
	return mcp.NewTool("create_learning",
		mcp.WithDescription("Create a new learning artifact linked to a session. Creates a learning node with 'learned_from' edge to session and 'references' edges to subareas/projects."),
		mcp.WithString("session_slug",
			mcp.Required(),
			mcp.Description("Slug of the session this learning came from"),
		),
		mcp.WithString("content",
			mcp.Required(),
			mcp.Description("Content of the learning"),
		),
		mcp.WithArray("subarea_slugs",
			mcp.Description("List of subarea slugs to tag this learning with"),
		),
		mcp.WithArray("project_slugs",
			mcp.Description("List of project slugs to tag this learning with"),
		),
	)
}

func listSessionsTool() mcp.Tool {
	return mcp.NewTool("list_sessions",
		mcp.WithDescription("List sessions, optionally filtered by project slug."),
		mcp.WithString("project_slug",
			mcp.Description("Filter sessions by project slug"),
		),
	)
}

func listLearningsTool() mcp.Tool {
	return mcp.NewTool("list_learnings",
		mcp.WithDescription("List learnings, optionally filtered by subarea or project slug."),
		mcp.WithString("subarea_slug",
			mcp.Description("Filter learnings by subarea slug"),
		),
		mcp.WithString("project_slug",
			mcp.Description("Filter learnings by project slug"),
		),
	)
}

func linkResourceTool() mcp.Tool {
	return mcp.NewTool("link_resource",
		mcp.WithDescription("Add a 'references' edge between an existing resource/node and a subarea. Use to tag resources with subareas."),
		mcp.WithString("resource_id",
			mcp.Required(),
			mcp.Description("Engram ID of the resource to tag"),
		),
		mcp.WithString("subarea_slug",
			mcp.Required(),
			mcp.Description("Slug of the subarea to link to"),
		),
	)
}

func listPersonTool() mcp.Tool {
	return mcp.NewTool("list_person",
		mcp.WithDescription("Get the user's person node from the knowledge graph."),
	)
}

func syncObsidianTool() mcp.Tool {
	return mcp.NewTool("sync_obsidian",
		mcp.WithDescription("Export the complete Marrow knowledge graph as markdown files with YAML frontmatter and wikilinks for Obsidian. Creates a Marrow/ directory structure under the specified vault path."),
		mcp.WithString("vault_path",
			mcp.Required(),
			mcp.Description("Absolute path to the Obsidian vault root. The graph will be exported to VaultPath/Marrow/."),
		),
	)
}
