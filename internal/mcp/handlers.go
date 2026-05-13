package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/Nick-2455/marrow/internal/app"
	"github.com/Nick-2455/marrow/internal/domain"
)

// Deps provides the dependencies needed by MCP handlers.
type Deps struct {
	Engram domain.EngramClient
	Store  domain.ResourceStore
}

// NewDeps creates handler dependencies from app.Deps.
func NewDeps(d *app.Deps) *Deps {
	return &Deps{
		Engram: d.Engram,
		Store:  d.Store,
	}
}

// global deps — set by StartServer before handlers are called
var handlerDeps *Deps

// handleSearch searches for resources with cache fallback.
func handleSearch(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, err := req.RequireString("query")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Try cache first
	cached, hit, cerr := handlerDeps.Store.GetCachedSearch(ctx, query)
	if cerr != nil {
		return mcp.NewToolResultError(fmt.Sprintf("cache error: %v", cerr)), nil
	}

	// Try Engram
	resources, err := handlerDeps.Engram.SearchResources(ctx, query)
	if err == nil {
		// Success — update cache
		_ = handlerDeps.Store.CacheSearch(ctx, query, resources)
		return resultFromResources(resources, false)
	}

	// Engram failed — fall back to cache
	if hit && len(cached) > 0 {
		return resultFromResources(cached, true)
	}

	return mcp.NewToolResultText(fmt.Sprintf(
		"Search returned no results. Engram error: %v", err,
	)), nil
}

// handleAddResource creates a new resource in Engram and sets triage position.
func handleAddResource(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	rawURL, err := req.RequireString("url")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Validate URL
	if _, perr := url.ParseRequestURI(rawURL); perr != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid URL: %v", perr)), nil
	}

	title := req.GetString("title", "")
	content := req.GetString("content", "")

	resource := domain.Resource{
		URL:     rawURL,
		Title:   title,
		Content: content,
		Bucket:  domain.BucketInbox,
	}

	// Create in Engram first
	id, err := handlerDeps.Engram.CreateResource(ctx, resource)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create resource in Engram: %v", err)), nil
	}

	// Set local triage position
	pos := domain.TriagePosition{
		ResourceID: id,
		Bucket:     domain.BucketInbox,
	}
	if err := handlerDeps.Store.SetTriagePosition(ctx, pos); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("created in Engram (id=%s) but failed to set local triage: %v", id, err)), nil
	}

	// Invalidate search cache so new resource appears immediately
	_ = handlerDeps.Store.InvalidateSearchCache(ctx)

	result := map[string]any{
		"id":     id,
		"bucket": string(domain.BucketInbox),
		"url":    rawURL,
	}
	data, _ := json.Marshal(result)
	return mcp.NewToolResultText(string(data)), nil
}

// handleGetRoadmap returns all resources grouped by bucket.
func handleGetRoadmap(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Get all triage positions from local SQLite
	positions, err := handlerDeps.Store.GetAllTriagePositions(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to load triage positions: %v", err)), nil
	}

	roadmap := make(map[domain.Bucket][]domain.Resource)
	for _, b := range domain.AllBuckets() {
		roadmap[b] = []domain.Resource{}
	}

	for _, pos := range positions {
		r := domain.Resource{
			ID:     pos.ResourceID,
			Bucket: pos.Bucket,
		}
		// Try to get full resource from Engram
		full, err := handlerDeps.Engram.GetResource(ctx, pos.ResourceID)
		if err == nil {
			r = full
		}
		roadmap[r.Bucket] = append(roadmap[r.Bucket], r)
	}

	data, err := json.MarshalIndent(roadmap, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal roadmap: %v", err)), nil
	}

	return mcp.NewToolResultText(string(data)), nil
}

// handleTriage moves a resource between buckets with rollback on Engram failure.
func handleTriage(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireString("id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	bucketStr, err := req.RequireString("bucket")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	bucket := domain.Bucket(bucketStr)
	if !bucket.Valid() {
		return mcp.NewToolResultError(fmt.Sprintf("invalid bucket: %q (must be inbox, active, later, or archived)", bucketStr)), nil
	}

	// Get current position for rollback
	oldPos, err := handlerDeps.Store.GetTriagePosition(ctx, id)
	if err != nil && err != domain.ErrTriageNotFound {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get current triage position: %v", err)), nil
	}

	// Update SQLite first
	newPos := domain.TriagePosition{
		ResourceID: id,
		Bucket:     bucket,
	}
	if err := handlerDeps.Store.SetTriagePosition(ctx, newPos); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to update local triage: %v", err)), nil
	}

	// Update Engram
	if err := handlerDeps.Engram.UpdateResource(ctx, id, map[string]any{
		"bucket": string(bucket),
	}); err != nil {
		// Rollback SQLite
		if oldPos.ResourceID != "" {
			_ = handlerDeps.Store.SetTriagePosition(ctx, oldPos)
		}
		return mcp.NewToolResultError(fmt.Sprintf("failed to update Engram (rolled back local): %v", err)), nil
	}

	result := map[string]any{
		"id":     id,
		"bucket": string(bucket),
	}
	data, _ := json.Marshal(result)
	return mcp.NewToolResultText(string(data)), nil
}

// resultFromResources formats a resource list as JSON for MCP response.
func resultFromResources(resources []domain.Resource, degraded bool) (*mcp.CallToolResult, error) {
	result := map[string]any{
		"results":    resources,
		"count":      len(resources),
		"degraded":   degraded,
	}
	if degraded {
		result["warning"] = "Engram unreachable — showing cached results"
	}

	data, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal results: %v", err)), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}

// StartServer starts the MCP stdio server with the given dependencies.
// It blocks until the server exits (e.g., on stdin EOF or error).
func StartServer(deps *Deps) error {
	handlerDeps = deps
	s := NewServer()
	return server.ServeStdio(s)
}
