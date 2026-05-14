package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/Nick-2455/marrow/internal/domain"
)

// handleListDomains returns all domains with their nested subareas.
func handleListDomains(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tree, err := handlerDeps.GraphStore.GetDomainTree(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to load domain tree: %v", err)), nil
	}

	type subareaOut struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	}
	type domainOut struct {
		Name     string        `json:"name"`
		Slug     string        `json:"slug"`
		Subareas []subareaOut  `json:"subareas"`
	}

	result := make([]domainOut, 0, len(tree))
	for _, dws := range tree {
		d := domainOut{
			Name:     dws.Domain.Title,
			Slug:     domain.Slugify(dws.Domain.Title),
			Subareas: make([]subareaOut, 0, len(dws.Subareas)),
		}
		for _, s := range dws.Subareas {
			d.Subareas = append(d.Subareas, subareaOut{
				Name: s.Title,
				Slug: domain.Slugify(s.Title),
			})
		}
		result = append(result, d)
	}

	data, _ := json.Marshal(result)
	return mcp.NewToolResultText(string(data)), nil
}

// handleListProjects returns all projects (active and inactive) with subarea links.
func handleListProjects(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	nodes, err := handlerDeps.GraphStore.ListNodesByType(ctx, domain.NodeTypeProject)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to load projects: %v", err)), nil
	}

	type projectOut struct {
		Name        string   `json:"name"`
		Slug        string   `json:"slug"`
		Description string   `json:"description"`
		Active      bool     `json:"active"`
		Subareas    []string `json:"subareas"`
	}

	result := make([]projectOut, 0, len(nodes))
	for _, n := range nodes {
		// Enrich with subarea links via "applies_to" edges
		edges, err := handlerDeps.GraphStore.GetEdges(ctx, n.EngramID, "from")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to load subarea links for %s: %v", n.Title, err)), nil
		}

		subareaSlugs := make([]string, 0, len(edges))
		for _, e := range edges {
			if e.Label == domain.EdgeAppliesTo {
				subNode, err := handlerDeps.GraphStore.GetNode(ctx, e.ToID)
				if err == nil {
					subareaSlugs = append(subareaSlugs, domain.Slugify(subNode.Title))
				}
			}
		}

		result = append(result, projectOut{
			Name:   n.Title,
			Slug:   domain.Slugify(n.Title),
			Active: n.Active,
			Subareas: subareaSlugs,
		})
	}

	data, _ := json.Marshal(result)
	return mcp.NewToolResultText(string(data)), nil
}

// handleCreateDomain creates a new domain in Engram + SQLite.
func handleCreateDomain(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := req.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	slug := domain.Slugify(name)
	if slug == "" {
		return mcp.NewToolResultError("invalid domain name: must contain alphanumeric characters"), nil
	}

	description := req.GetString("description", "")

	// Create in Engram first (source of truth)
	engramID, err := handlerDeps.Engram.SaveNode(ctx, string(domain.NodeTypeDomain), name, map[string]any{
		"name":        name,
		"slug":        slug,
		"description": description,
		"active":      true,
	}, "domain/"+slug)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create domain in Engram: %v", err)), nil
	}

	// Cache in SQLite
	if err := handlerDeps.GraphStore.UpsertNode(ctx, domain.GraphNode{
		EngramID: engramID,
		NodeType: domain.NodeTypeDomain,
		Title:    name,
		Active:   true,
	}); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("created in Engram (id=%s) but failed to cache in SQLite: %v", engramID, err)), nil
	}

	result := map[string]any{
		"engram_id": engramID,
		"name":      name,
		"slug":      slug,
	}
	data, _ := json.Marshal(result)
	return mcp.NewToolResultText(string(data)), nil
}

// handleCreateSubarea creates a subarea under a domain.
func handleCreateSubarea(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := req.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	domainSlug, err := req.RequireString("domain_slug")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	slug := domain.Slugify(name)
	if slug == "" {
		return mcp.NewToolResultError("invalid subarea name: must contain alphanumeric characters"), nil
	}

	// Find domain by title match (we need to search for the domain)
	domains, err := handlerDeps.GraphStore.ListNodesByType(ctx, domain.NodeTypeDomain)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to load domains: %v", err)), nil
	}

	var domainNode *domain.GraphNode
	for i := range domains {
		if domain.Slugify(domains[i].Title) == domainSlug {
			domainNode = &domains[i]
			break
		}
	}
	if domainNode == nil {
		return mcp.NewToolResultError(fmt.Sprintf("domain not found: %q", domainSlug)), nil
	}

	// Create in Engram first
	topicKey := "subarea/" + domainSlug + "/" + slug
	engramID, err := handlerDeps.Engram.SaveNode(ctx, string(domain.NodeTypeSubarea), name, map[string]any{
		"name":      name,
		"slug":      slug,
		"domain_id": domainNode.EngramID,
		"active":    true,
	}, topicKey)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create subarea in Engram: %v", err)), nil
	}

	// Cache in SQLite
	if err := handlerDeps.GraphStore.UpsertNode(ctx, domain.GraphNode{
		EngramID: engramID,
		NodeType: domain.NodeTypeSubarea,
		Title:    name,
		Active:   true,
	}); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("created in Engram (id=%s) but failed to cache in SQLite: %v", engramID, err)), nil
	}

	// Add contains edge: Domain → Subarea
	if err := handlerDeps.GraphStore.AddEdge(ctx, domainNode.EngramID, engramID, domain.EdgeContains); err != nil {
		if err == domain.ErrDuplicateNode {
			return mcp.NewToolResultError(fmt.Sprintf("subarea %q already exists under domain %q", name, domainSlug)), nil
		}
		return mcp.NewToolResultError(fmt.Sprintf("created subarea but failed to link to domain: %v", err)), nil
	}

	result := map[string]any{
		"engram_id":    engramID,
		"name":         name,
		"slug":         slug,
		"domain_slug":  domainSlug,
	}
	data, _ := json.Marshal(result)
	return mcp.NewToolResultText(string(data)), nil
}

// handleCreateProject creates a new project in Engram + SQLite.
func handleCreateProject(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := req.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	slug := domain.Slugify(name)
	if slug == "" {
		return mcp.NewToolResultError("invalid project name: must contain alphanumeric characters"), nil
	}

	description := req.GetString("description", "")

	// Create in Engram first
	engramID, err := handlerDeps.Engram.SaveNode(ctx, string(domain.NodeTypeProject), name, map[string]any{
		"name":        name,
		"slug":        slug,
		"description": description,
		"active":      true,
	}, "project/"+slug)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create project in Engram: %v", err)), nil
	}

	// Cache in SQLite
	if err := handlerDeps.GraphStore.UpsertNode(ctx, domain.GraphNode{
		EngramID: engramID,
		NodeType: domain.NodeTypeProject,
		Title:    name,
		Active:   true,
	}); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("created in Engram (id=%s) but failed to cache in SQLite: %v", engramID, err)), nil
	}

	result := map[string]any{
		"engram_id": engramID,
		"name":      name,
		"slug":      slug,
	}
	data, _ := json.Marshal(result)
	return mcp.NewToolResultText(string(data)), nil
}

// handleLinkProject links a project to a subarea via applies_to edge.
func handleLinkProject(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectSlug, err := req.RequireString("project_slug")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	subareaSlug, err := req.RequireString("subarea_slug")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Find project node
	projects, err := handlerDeps.GraphStore.ListNodesByType(ctx, domain.NodeTypeProject)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to load projects: %v", err)), nil
	}

	var projectID string
	for _, p := range projects {
		if domain.Slugify(p.Title) == projectSlug {
			projectID = p.EngramID
			break
		}
	}
	if projectID == "" {
		return mcp.NewToolResultError(fmt.Sprintf("project not found: %q", projectSlug)), nil
	}

	// Find subarea node — need to search all subareas
	subareas, err := handlerDeps.GraphStore.ListNodesByType(ctx, domain.NodeTypeSubarea)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to load subareas: %v", err)), nil
	}

	var subareaID string
	for _, s := range subareas {
		if domain.Slugify(s.Title) == subareaSlug {
			subareaID = s.EngramID
			break
		}
	}
	if subareaID == "" {
		return mcp.NewToolResultError(fmt.Sprintf("subarea not found: %q", subareaSlug)), nil
	}

	// Add applies_to edge: Project → Subarea
	if err := handlerDeps.GraphStore.AddEdge(ctx, projectID, subareaID, domain.EdgeAppliesTo); err != nil {
		if err == domain.ErrDuplicateNode {
			return mcp.NewToolResultError(fmt.Sprintf("project %q is already linked to subarea %q", projectSlug, subareaSlug)), nil
		}
		return mcp.NewToolResultError(fmt.Sprintf("failed to link project to subarea: %v", err)), nil
	}

	result := map[string]any{
		"project_slug":  projectSlug,
		"subarea_slug":  subareaSlug,
		"edge_label":    string(domain.EdgeAppliesTo),
	}
	data, _ := json.Marshal(result)
	return mcp.NewToolResultText(string(data)), nil
}

// handleToggleProject toggles a project's active state.
func handleToggleProject(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectSlug, err := req.RequireString("project_slug")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Find project node
	projects, err := handlerDeps.GraphStore.ListNodesByType(ctx, domain.NodeTypeProject)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to load projects: %v", err)), nil
	}

	var node *domain.GraphNode
	for i := range projects {
		if domain.Slugify(projects[i].Title) == projectSlug {
			node = &projects[i]
			break
		}
	}
	if node == nil {
		return mcp.NewToolResultError(fmt.Sprintf("project not found: %q", projectSlug)), nil
	}

	newActive := !node.Active

	// Update SQLite cache
	if err := handlerDeps.GraphStore.UpsertNode(ctx, domain.GraphNode{
		EngramID: node.EngramID,
		NodeType: domain.NodeTypeProject,
		Title:    node.Title,
		Active:   newActive,
	}); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to update project: %v", err)), nil
	}

	// Update Engram
	if err := handlerDeps.Engram.UpdateNode(ctx, node.EngramID, map[string]any{
		"active": newActive,
	}); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("updated SQLite but failed to update Engram: %v", err)), nil
	}

	action := "activated"
	if !newActive {
		action = "deactivated"
	}

	result := map[string]any{
		"name":   node.Title,
		"slug":   projectSlug,
		"active": newActive,
		"action": action,
	}
	data, _ := json.Marshal(result)
	return mcp.NewToolResultText(string(data)), nil
}
