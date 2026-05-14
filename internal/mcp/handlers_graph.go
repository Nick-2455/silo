package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/Nick-2455/marrow/internal/domain"
	"github.com/Nick-2455/marrow/internal/obsidian"
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

// handleCreateSession creates a session linked to a project.
func handleCreateSession(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectSlug, err := req.RequireString("project_slug")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	description, err := req.RequireString("description")
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

	slug := domain.Slugify(description)
	if slug == "" {
		slug = "session"
	}
	sessionID := "session/" + slug

	// Create in Engram first
	engramID, err := handlerDeps.Engram.SaveNode(ctx, string(domain.NodeTypeSession), description, map[string]any{
		"description": description,
		"project_id":  projectID,
		"active":      true,
	}, "session/"+slug)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create session in Engram: %v", err)), nil
	}

	// Cache in SQLite
	if err := handlerDeps.GraphStore.UpsertNode(ctx, domain.GraphNode{
		EngramID: engramID,
		NodeType: domain.NodeTypeSession,
		Title:    description,
		Active:   true,
	}); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("created in Engram (id=%s) but failed to cache in SQLite: %v", engramID, err)), nil
	}

	// Add worked_on edge: Project → Session
	if err := handlerDeps.GraphStore.AddEdge(ctx, projectID, engramID, domain.EdgeWorkedOn); err != nil {
		if err == domain.ErrDuplicateNode {
			return mcp.NewToolResultError(fmt.Sprintf("session already linked to project %q", projectSlug)), nil
		}
		return mcp.NewToolResultError(fmt.Sprintf("created session but failed to link to project: %v", err)), nil
	}

	result := map[string]any{
		"engram_id":    engramID,
		"id":           sessionID,
		"description":  description,
		"project_slug": projectSlug,
	}
	data, _ := json.Marshal(result)
	return mcp.NewToolResultText(string(data)), nil
}

// handleCreateLearning creates a learning linked to a session and subareas.
func handleCreateLearning(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionSlug, err := req.RequireString("session_slug")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	content, err := req.RequireString("content")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Find session node
	sessions, err := handlerDeps.GraphStore.ListNodesByType(ctx, domain.NodeTypeSession)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to load sessions: %v", err)), nil
	}

	var sessionID string
	for _, s := range sessions {
		if domain.Slugify(s.Title) == sessionSlug {
			sessionID = s.EngramID
			break
		}
	}
	if sessionID == "" {
		return mcp.NewToolResultError(fmt.Sprintf("session not found: %q", sessionSlug)), nil
	}

	slug := domain.Slugify(content)
	if slug == "" {
		slug = "learning"
	}
	if len(slug) > 40 {
		slug = slug[:40]
	}
	learningID := "learning/" + slug

	// Create in Engram first
	engramID, err := handlerDeps.Engram.SaveNode(ctx, string(domain.NodeTypeLearning), content[:min(len(content), 80)], map[string]any{
		"content":     content,
		"session_id":  sessionID,
		"active":      true,
	}, "learning/"+slug)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create learning in Engram: %v", err)), nil
	}

	// Cache in SQLite
	if err := handlerDeps.GraphStore.UpsertNode(ctx, domain.GraphNode{
		EngramID: engramID,
		NodeType: domain.NodeTypeLearning,
		Title:    content,
		Active:   true,
	}); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("created in Engram (id=%s) but failed to cache in SQLite: %v", engramID, err)), nil
	}

	// Add learned_from edge: Learning → Session
	if err := handlerDeps.GraphStore.AddEdge(ctx, engramID, sessionID, domain.EdgeLearnedFrom); err != nil {
		if err != domain.ErrDuplicateNode {
			return mcp.NewToolResultError(fmt.Sprintf("created learning but failed to link to session: %v", err)), nil
		}
	}

	// Add references edges to subareas
	if args, ok := req.Params.Arguments.(map[string]any); ok {
		if subSlugsVal, exists := args["subarea_slugs"]; exists {
			if subSlugs, ok := subSlugsVal.([]any); ok {
				for _, item := range subSlugs {
					if subSlug, ok := item.(string); ok && subSlug != "" {
						subareas, err := handlerDeps.GraphStore.ListNodesByType(ctx, domain.NodeTypeSubarea)
						if err != nil {
							continue
						}
						for _, sa := range subareas {
							if domain.Slugify(sa.Title) == subSlug {
								_ = handlerDeps.GraphStore.AddEdge(ctx, engramID, sa.EngramID, domain.EdgeReferences)
								break
							}
						}
					}
				}
			}
		}
	}

	// Add references edges to projects
	if args, ok := req.Params.Arguments.(map[string]any); ok {
		if projSlugsVal, exists := args["project_slugs"]; exists {
			if projSlugs, ok := projSlugsVal.([]any); ok {
				for _, item := range projSlugs {
					if projSlug, ok := item.(string); ok && projSlug != "" {
						projects, err := handlerDeps.GraphStore.ListNodesByType(ctx, domain.NodeTypeProject)
						if err != nil {
							continue
						}
						for _, p := range projects {
							if domain.Slugify(p.Title) == projSlug {
								_ = handlerDeps.GraphStore.AddEdge(ctx, engramID, p.EngramID, domain.EdgeReferences)
								break
							}
						}
					}
				}
			}
		}
	}

	result := map[string]any{
		"engram_id":    engramID,
		"id":           learningID,
		"content":      content,
		"session_slug": sessionSlug,
	}
	data, _ := json.Marshal(result)
	return mcp.NewToolResultText(string(data)), nil
}

// handleListSessions returns sessions, optionally filtered by project.
func handleListSessions(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectSlug := req.GetString("project_slug", "")

	type sessionOut struct {
		ID          string `json:"id"`
		ProjectID   string `json:"project_id"`
		Description string `json:"description"`
		CreatedAt   string `json:"created_at"`
	}

	var allSessions []sessionOut

	if projectSlug != "" {
		// Find project node
		projects, err := handlerDeps.GraphStore.ListNodesByType(ctx, domain.NodeTypeProject)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to load projects: %v", err)), nil
		}
		for _, p := range projects {
			if domain.Slugify(p.Title) == projectSlug {
				sessions, err := handlerDeps.GraphStore.ListSessions(ctx, p.EngramID)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to load sessions: %v", err)), nil
				}
				for _, s := range sessions {
					allSessions = append(allSessions, sessionOut{
						ID:          s.ID,
						ProjectID:   s.ProjectID,
						Description: s.Description,
						CreatedAt:   s.CreatedAt.Format(time.RFC3339),
					})
				}
				break
			}
		}
	} else {
		// List all sessions
		nodes, err := handlerDeps.GraphStore.ListNodesByType(ctx, domain.NodeTypeSession)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to load sessions: %v", err)), nil
		}
		for _, n := range nodes {
			if !n.Active {
				continue
			}
			allSessions = append(allSessions, sessionOut{
				ID:          n.EngramID,
				Description: n.Title,
			})
		}
	}

	data, _ := json.Marshal(allSessions)
	return mcp.NewToolResultText(string(data)), nil
}

// handleListLearnings returns learnings, optionally filtered by subarea or project.
func handleListLearnings(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	subareaSlug := req.GetString("subarea_slug", "")
	projectSlug := req.GetString("project_slug", "")

	var subareaID string
	if subareaSlug != "" {
		subareas, err := handlerDeps.GraphStore.ListNodesByType(ctx, domain.NodeTypeSubarea)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to load subareas: %v", err)), nil
		}
		for _, sa := range subareas {
			if domain.Slugify(sa.Title) == subareaSlug {
				subareaID = sa.EngramID
				break
			}
		}
		if subareaID == "" {
			return mcp.NewToolResultError(fmt.Sprintf("subarea not found: %q", subareaSlug)), nil
		}
	}

	learnings, err := handlerDeps.GraphStore.ListLearnings(ctx, subareaID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to load learnings: %v", err)), nil
	}

	// Filter by project if specified
	if projectSlug != "" {
		projects, err := handlerDeps.GraphStore.ListNodesByType(ctx, domain.NodeTypeProject)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to load projects: %v", err)), nil
		}
		var targetProjectID string
		for _, p := range projects {
			if domain.Slugify(p.Title) == projectSlug {
				targetProjectID = p.EngramID
				break
			}
		}
		if targetProjectID == "" {
			return mcp.NewToolResultError(fmt.Sprintf("project not found: %q", projectSlug)), nil
		}

		filtered := make([]domain.Learning, 0, len(learnings))
		for _, l := range learnings {
			for _, pid := range l.ProjectIDs {
				if pid == targetProjectID {
					filtered = append(filtered, l)
					break
				}
			}
		}
		learnings = filtered
	}

	type learningOut struct {
		ID         string   `json:"id"`
		Content    string   `json:"content"`
		SessionID  string   `json:"session_id"`
		SubareaIDs []string `json:"subarea_ids"`
		CreatedAt  string   `json:"created_at"`
	}

	result := make([]learningOut, 0, len(learnings))
	for _, l := range learnings {
		preview := l.Content
		if len(preview) > 200 {
			preview = preview[:197] + "..."
		}
		result = append(result, learningOut{
			ID:         l.ID,
			Content:    preview,
			SessionID:  l.SessionID,
			SubareaIDs: l.SubareaIDs,
			CreatedAt:  l.CreatedAt.Format(time.RFC3339),
		})
	}

	data, _ := json.Marshal(result)
	return mcp.NewToolResultText(string(data)), nil
}

// handleLinkResource adds a references edge between a resource (learning node) and a subarea.
func handleLinkResource(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	resourceID, err := req.RequireString("resource_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	subareaSlug, err := req.RequireString("subarea_slug")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Find subarea node
	subareas, err := handlerDeps.GraphStore.ListNodesByType(ctx, domain.NodeTypeSubarea)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to load subareas: %v", err)), nil
	}

	var subareaID string
	for _, sa := range subareas {
		if domain.Slugify(sa.Title) == subareaSlug {
			subareaID = sa.EngramID
			break
		}
	}
	if subareaID == "" {
		return mcp.NewToolResultError(fmt.Sprintf("subarea not found: %q", subareaSlug)), nil
	}

	// Verify resource exists
	_, err = handlerDeps.GraphStore.GetNode(ctx, resourceID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("resource not found: %q", resourceID)), nil
	}

	// Add references edge
	if err := handlerDeps.GraphStore.AddEdge(ctx, resourceID, subareaID, domain.EdgeReferences); err != nil {
		if err == domain.ErrDuplicateNode {
			return mcp.NewToolResultError(fmt.Sprintf("resource %q already linked to subarea %q", resourceID, subareaSlug)), nil
		}
		return mcp.NewToolResultError(fmt.Sprintf("failed to link resource to subarea: %v", err)), nil
	}

	result := map[string]any{
		"resource_id":  resourceID,
		"subarea_slug": subareaSlug,
		"edge_label":   string(domain.EdgeReferences),
	}
	data, _ := json.Marshal(result)
	return mcp.NewToolResultText(string(data)), nil
}

// handleListPerson returns the user's person node.
func handleListPerson(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	nodes, err := handlerDeps.GraphStore.ListNodesByType(ctx, domain.NodeTypePerson)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to load person: %v", err)), nil
	}

	if len(nodes) == 0 {
		return mcp.NewToolResultText(`{"message": "no person node found"}`), nil
	}

	// Return the first active person
	for _, n := range nodes {
		if n.Active {
			result := map[string]any{
				"engram_id": n.EngramID,
				"name":      n.Title,
				"active":    n.Active,
			}
			data, _ := json.Marshal(result)
			return mcp.NewToolResultText(string(data)), nil
		}
	}

	return mcp.NewToolResultText(`{"message": "no active person found"}`), nil
}

// handleSyncObsidian exports the complete graph as markdown files for Obsidian.
func handleSyncObsidian(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	vaultPath := req.GetString("vault_path", "")
	if vaultPath == "" {
		return mcp.NewToolResultError("no vault path configured"), nil
	}

	syncer := &obsidian.Syncer{
		Store:     handlerDeps.GraphStore,
		VaultPath: vaultPath,
	}

	report, err := syncer.SyncAll(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("obsidian sync failed: %v", err)), nil
	}

	data, _ := json.Marshal(report)
	return mcp.NewToolResultText(string(data)), nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
