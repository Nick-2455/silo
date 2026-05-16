package mcp

import (
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	serverName    = "silo"
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
	s.AddTool(discoverProjectTool(), handleDiscoverProject)

	s.AddTool(discoverProjectTool(), handleDiscoverProject)
	s.AddTool(importProjectTool(), handleImportProject)

	// Session and Learning tools
	s.AddTool(createSessionTool(), handleCreateSession)
	s.AddTool(createLearningTool(), handleCreateLearning)
	s.AddTool(listSessionsTool(), handleListSessions)
	s.AddTool(listLearningsTool(), handleListLearnings)
	s.AddTool(linkResourceTool(), handleLinkResource)
	s.AddTool(listPersonTool(), handleListPerson)
	s.AddTool(syncObsidianTool(), handleSyncObsidian)

	// MVP bridge tools
	s.AddTool(readFromEngramTool(), handleReadFromEngram)
	s.AddTool(syncToObsidianTool(), handleSyncToObsidian)
	s.AddTool(searchVaultTool(), handleSearchVault)
	s.AddTool(createOrUpdateNoteTool(), handleCreateOrUpdateNote)
	s.AddTool(getKnowledgeContextTool(), handleGetKnowledgeContext)

	return s
}

// --- Resource tools ---

func searchTool() mcp.Tool {
	return mcp.NewTool("search",
		mcp.WithDescription("[DEPRECATED] Search for resources in the legacy knowledge graph. Prefer read_from_engram for the MVP bridge."),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Search query for FTS5 search"),
		),
	)
}

func addResourceTool() mcp.Tool {
	return mcp.NewTool("add_resource",
		mcp.WithDescription("[DEPRECATED] Add a new resource to the legacy knowledge graph. Creates an Engram observation in the inbox bucket."),
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
		mcp.WithDescription("[DEPRECATED] Get the legacy roadmap with resources grouped by bucket (inbox, active, later, archived)."),
	)
}

func triageTool() mcp.Tool {
	return mcp.NewTool("triage",
		mcp.WithDescription("[DEPRECATED] Move a legacy resource between buckets. Updates both local state and Engram."),
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
		mcp.WithDescription("[DEPRECATED] List legacy domains with their nested subareas. Hierarchical knowledge taxonomy view."),
	)
}

func listProjectsTool() mcp.Tool {
	return mcp.NewTool("list_projects",
		mcp.WithDescription("[DEPRECATED] List legacy projects (active and inactive) with their linked subareas."),
	)
}

func createDomainTool() mcp.Tool {
	return mcp.NewTool("create_domain",
		mcp.WithDescription("[DEPRECATED] Create a legacy knowledge domain. Stores in Engram and caches in SQLite."),
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
		mcp.WithDescription("[DEPRECATED] Create a legacy subarea under an existing domain. Adds a 'contains' edge from domain to subarea."),
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
		mcp.WithDescription("[DEPRECATED] Create a legacy project. Stores in Engram and caches in SQLite. Use link_project to connect to subareas."),
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
		mcp.WithDescription("[DEPRECATED] Link a legacy project to a subarea. Creates an 'applies_to' edge from project to subarea."),
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
		mcp.WithDescription("[DEPRECATED] Toggle a legacy project between active and inactive states. Updates both Engram and SQLite."),
		mcp.WithString("project_slug",
			mcp.Required(),
			mcp.Description("Slug of the project to toggle"),
		),
	)
}

func discoverProjectTool() mcp.Tool {
	return mcp.NewTool("discover_project",
		mcp.WithDescription("[DEPRECATED] Search Engram for pre-existing observations under a project (legacy import workflow)."),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the project to discover (e.g. 'cortex', 'blacksight')"),
		),
	)
}

func importProjectTool() mcp.Tool {
	return mcp.NewTool("import_project",
		mcp.WithDescription("[DEPRECATED] Auto-import Engram observations as legacy Silo graph nodes (sessions and learnings). Idempotent."),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the existing Silo project to import observations for"),
		),
	)
}

// --- Session and Learning tools ---

func createSessionTool() mcp.Tool {
	return mcp.NewTool("create_session",
		mcp.WithDescription("[DEPRECATED] Create a legacy session linked to a project. Creates a session node and a 'worked_on' edge from project to session."),
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
		mcp.WithDescription("[DEPRECATED] Create a legacy learning artifact linked to a session. Creates a learning node with 'learned_from' edge to session and 'references' edges to subareas/projects."),
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
			mcp.WithStringItems(),
		),
		mcp.WithArray("project_slugs",
			mcp.Description("List of project slugs to tag this learning with"),
			mcp.WithStringItems(),
		),
	)
}

func listSessionsTool() mcp.Tool {
	return mcp.NewTool("list_sessions",
		mcp.WithDescription("[DEPRECATED] List legacy sessions, optionally filtered by project slug."),
		mcp.WithString("project_slug",
			mcp.Description("Filter sessions by project slug"),
		),
	)
}

func listLearningsTool() mcp.Tool {
	return mcp.NewTool("list_learnings",
		mcp.WithDescription("[DEPRECATED] List legacy learnings, optionally filtered by subarea or project slug."),
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
		mcp.WithDescription("[DEPRECATED] Add a 'references' edge between an existing legacy resource/node and a subarea."),
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
		mcp.WithDescription("[DEPRECATED] Get the user's legacy person node from the knowledge graph."),
	)
}

func syncObsidianTool() mcp.Tool {
	return mcp.NewTool("sync_obsidian",
		mcp.WithDescription("[DEPRECATED] Export the legacy Silo knowledge graph as Markdown files for Obsidian. Prefer sync_to_obsidian for the MVP bridge."),
		mcp.WithString("vault_path",
			mcp.Required(),
			mcp.Description("Absolute path to the Obsidian vault root. The graph will be exported to VaultPath/Silo/."),
		),
	)
}
