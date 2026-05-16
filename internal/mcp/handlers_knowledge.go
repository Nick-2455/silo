package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/Nick-2455/silo/internal/knowledge"
	"github.com/Nick-2455/silo/internal/knowledge/notemodel"
)

// KnowledgeService is the subset of knowledge.Service that the MCP handlers use.
// Defined as an interface so tests can inject a fake service.
type KnowledgeService interface {
	ReadKnowledge(ctx context.Context, q knowledge.EngramQuery) ([]knowledge.KnowledgeItem, error)
	SyncToObsidian(ctx context.Context, req knowledge.SyncRequest) (*knowledge.SyncReport, error)
	SearchVault(ctx context.Context, vaultPath, query string, limit int) ([]knowledge.NoteSearchResult, error)
	CreateOrUpdateNote(ctx context.Context, vaultPath string, note knowledge.Note) (knowledge.NoteWriteResult, error)
	GetKnowledgeContext(ctx context.Context, req knowledge.ContextRequest) (*knowledge.KnowledgeContext, error)
}

// VaultResolver returns the configured vault path when callers omit one.
type VaultResolver func() string

func defaultVaultResolver() string {
	if handlerDeps == nil {
		return ""
	}
	return handlerDeps.Config.ObsidianVaultPath
}

func resolveVaultPath(arg string) (string, error) {
	if arg != "" {
		return arg, nil
	}
	if handlerDeps != nil && handlerDeps.VaultPath != nil {
		if v := handlerDeps.VaultPath(); v != "" {
			return v, nil
		}
	}
	if path := defaultVaultResolver(); path != "" {
		return path, nil
	}
	return "", errors.New("vault path not configured: pass vault_path or set obsidian_vault_path in config")
}

func requireKnowledge() (KnowledgeService, error) {
	if handlerDeps == nil || handlerDeps.Knowledge == nil {
		return nil, errors.New("knowledge service is not configured")
	}
	return handlerDeps.Knowledge, nil
}

// --- Tool schemas ---

func readFromEngramTool() mcp.Tool {
	return mcp.NewTool("read_from_engram",
		mcp.WithDescription("Read knowledge items from Engram via mem_search and normalize them for agents."),
		mcp.WithString("query", mcp.Required(), mcp.Description("Search query for Engram")),
		mcp.WithString("project", mcp.Description("Optional Engram project filter")),
		mcp.WithString("type", mcp.Description("Optional Engram observation type")),
		mcp.WithNumber("limit", mcp.Description("Maximum number of items to return")),
	)
}

func syncToObsidianTool() mcp.Tool {
	return mcp.NewTool("sync_to_obsidian",
		mcp.WithDescription("Read Engram knowledge and write Markdown notes into the Obsidian vault."),
		mcp.WithString("query", mcp.Required(), mcp.Description("Engram search query for the notes to sync")),
		mcp.WithString("project", mcp.Description("Optional Engram project filter")),
		mcp.WithString("type", mcp.Description("Optional Engram observation type")),
		mcp.WithNumber("limit", mcp.Description("Maximum number of items to sync")),
		mcp.WithString("vault_path", mcp.Description("Vault path; defaults to obsidian_vault_path config")),
	)
}

func searchVaultTool() mcp.Tool {
	return mcp.NewTool("search_vault",
		mcp.WithDescription("Search Markdown files in an Obsidian vault."),
		mcp.WithString("query", mcp.Required(), mcp.Description("Plain-text query")),
		mcp.WithString("vault_path", mcp.Description("Vault path; defaults to obsidian_vault_path config")),
		mcp.WithNumber("limit", mcp.Description("Maximum number of matches to return")),
	)
}

func createOrUpdateNoteTool() mcp.Tool {
	return mcp.NewTool("create_or_update_note",
		mcp.WithDescription("Create or update a Markdown note in the Obsidian vault. "+
			"Pass type (concept | resource | roadmap | collection) to apply community note-model frontmatter defaults. "+
			"Pass kind to further classify the note (e.g. resource kind: book)."),
		mcp.WithString("title", mcp.Required(), mcp.Description("Note title used to derive a filename when no path is given")),
		mcp.WithString("content", mcp.Required(), mcp.Description("Markdown body")),
		mcp.WithString("path", mcp.Description("Relative path inside the vault; optional")),
		mcp.WithString("vault_path", mcp.Description("Vault path; defaults to obsidian_vault_path config")),
		mcp.WithBoolean("overwrite", mcp.Description("Overwrite existing file when the resolved target path already exists (default: false)")),
		mcp.WithString("type", mcp.Description("Community note type: concept, resource, roadmap, or collection")),
		mcp.WithString("kind", mcp.Description("Optional sub-category (e.g. book, article for resource; subject, project for collection)")),
	)
}

func listNoteTemplatesTool() mcp.Tool {
	return mcp.NewTool("list_note_templates",
		mcp.WithDescription("List available community note templates with their type, description, and frontmatter schema. "+
			"Use get_note_template to fetch the full Markdown body for a specific type."),
	)
}

func getNoteTemplateTool() mcp.Tool {
	return mcp.NewTool("get_note_template",
		mcp.WithDescription("Fetch the full Markdown body for a community note template. "+
			"Returns the raw template with frontmatter and placeholder body sections."),
		mcp.WithString("type", mcp.Required(), mcp.Description("Note type: concept, resource, roadmap, or collection")),
		mcp.WithString("kind", mcp.Description("Optional kind hint (informational only; does not change the returned body)")),
	)
}

func getKnowledgeContextTool() mcp.Tool {
	return mcp.NewTool("get_knowledge_context",
		mcp.WithDescription("Combine Engram knowledge and Obsidian vault matches into a knowledge context payload."),
		mcp.WithString("query", mcp.Required(), mcp.Description("Knowledge query")),
		mcp.WithString("project", mcp.Description("Optional Engram project filter")),
		mcp.WithString("vault_path", mcp.Description("Vault path; defaults to obsidian_vault_path config")),
		mcp.WithNumber("limit", mcp.Description("Maximum items per source")),
	)
}

// --- Handlers ---

func handleReadFromEngram(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	svc, err := requireKnowledge()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	query, err := req.RequireString("query")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	limit := int(req.GetFloat("limit", 0))
	items, err := svc.ReadKnowledge(ctx, knowledge.EngramQuery{
		Query:   query,
		Project: req.GetString("project", ""),
		Type:    req.GetString("type", ""),
		Limit:   limit,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("read_from_engram: %v", err)), nil
	}
	return jsonResult(map[string]any{
		"query": query,
		"items": items,
		"count": len(items),
	})
}

func handleSyncToObsidian(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	svc, err := requireKnowledge()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	query, err := req.RequireString("query")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	vaultPath, err := resolveVaultPath(req.GetString("vault_path", ""))
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	report, err := svc.SyncToObsidian(ctx, knowledge.SyncRequest{
		VaultPath: vaultPath,
		Engram: knowledge.EngramQuery{
			Query:   query,
			Project: req.GetString("project", ""),
			Type:    req.GetString("type", ""),
			Limit:   int(req.GetFloat("limit", 0)),
		},
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("sync_to_obsidian: %v", err)), nil
	}
	return jsonResult(report)
}

func handleSearchVault(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	svc, err := requireKnowledge()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	query, err := req.RequireString("query")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	vaultPath, err := resolveVaultPath(req.GetString("vault_path", ""))
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	limit := int(req.GetFloat("limit", 0))
	results, err := svc.SearchVault(ctx, vaultPath, query, limit)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("search_vault: %v", err)), nil
	}
	return jsonResult(map[string]any{
		"query":   query,
		"results": results,
		"count":   len(results),
	})
}

func handleCreateOrUpdateNote(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	svc, err := requireKnowledge()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	title, err := req.RequireString("title")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	content, err := req.RequireString("content")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	vaultPath, err := resolveVaultPath(req.GetString("vault_path", ""))
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	rawType := req.GetString("type", "")
	rawKind := req.GetString("kind", "")
	if rawType == "" && rawKind != "" {
		return mcp.NewToolResultError("kind is only valid when type is provided"), nil
	}
	if rawType != "" {
		if err := notemodel.Validate(notemodel.Type(rawType), rawKind); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
	}
	note := knowledge.Note{
		Title:   title,
		Path:    req.GetString("path", ""),
		Content: content,
		Overwrite: req.GetBool("overwrite", false),
		Type:      rawType,
		Kind:      rawKind,
	}
	result, err := svc.CreateOrUpdateNote(ctx, vaultPath, note)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("create_or_update_note: %v", err)), nil
	}
	return jsonResult(result)
}

func handleGetKnowledgeContext(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	svc, err := requireKnowledge()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	query, err := req.RequireString("query")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	vaultArg := req.GetString("vault_path", "")
	vaultPath, vaultErr := resolveVaultPath(vaultArg)
	if vaultErr != nil {
		vaultPath = ""
	}
	ctxResult, err := svc.GetKnowledgeContext(ctx, knowledge.ContextRequest{
		Query:     query,
		Project:   req.GetString("project", ""),
		VaultPath: vaultPath,
		Limit:     int(req.GetFloat("limit", 0)),
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("get_knowledge_context: %v", err)), nil
	}
	return jsonResult(ctxResult)
}

func handleListNoteTemplates(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tmpls := notemodel.Templates()
	// Return metadata only — body is omitted; use get_note_template for the body.
	type meta struct {
		Type              string                      `json:"type"`
		DefaultKind       string                      `json:"default_kind,omitempty"`
		Description       string                      `json:"description"`
		FrontmatterSchema notemodel.FrontmatterSchema `json:"frontmatter_schema"`
	}
	out := make([]meta, len(tmpls))
	for i, t := range tmpls {
		out[i] = meta{
			Type:              string(t.Type),
			DefaultKind:       t.DefaultKind,
			Description:       t.Description,
			FrontmatterSchema: t.FrontmatterSchema,
		}
	}
	return jsonResult(map[string]any{
		"templates": out,
		"count":     len(out),
	})
}

func handleGetNoteTemplate(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	rawType, err := req.RequireString("type")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	t := notemodel.Type(rawType)
	tmpl, ok := notemodel.Get(t)
	if !ok {
		return mcp.NewToolResultError(fmt.Sprintf(
			"unknown note type %q: valid types are concept, resource, roadmap, collection", rawType,
		)), nil
	}
	return jsonResult(map[string]any{
		"type": string(tmpl.Type),
		"kind": req.GetString("kind", tmpl.DefaultKind),
		"body": tmpl.Body(),
	})
}

func jsonResult(payload any) (*mcp.CallToolResult, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("marshal response: %v", err)), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}
