package store_test

import (
	"context"
	"testing"

	"github.com/Nick-2455/marrow/internal/domain"
)

func TestUpsertNode_InsertAndGet(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	node := domain.GraphNode{
		EngramID: "node-1",
		NodeType: domain.NodeTypeDomain,
		Title:    "Backend Development",
		Active:   true,
	}

	if err := s.UpsertNode(ctx, node); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	got, err := s.GetNode(ctx, "node-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.EngramID != "node-1" {
		t.Errorf("engram_id: got %q, want %q", got.EngramID, "node-1")
	}
	if got.NodeType != domain.NodeTypeDomain {
		t.Errorf("node_type: got %q, want %q", got.NodeType, domain.NodeTypeDomain)
	}
	if got.Title != "Backend Development" {
		t.Errorf("title: got %q, want %q", got.Title, "Backend Development")
	}
	if !got.Active {
		t.Error("active: got false, want true")
	}
	if got.CachedAt.IsZero() {
		t.Error("cached_at: expected non-zero time")
	}
}

func TestUpsertNode_Update(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	node := domain.GraphNode{
		EngramID: "node-1",
		NodeType: domain.NodeTypeDomain,
		Title:    "Backend",
		Active:   true,
	}
	if err := s.UpsertNode(ctx, node); err != nil {
		t.Fatalf("initial upsert: %v", err)
	}

	// Update the title
	node.Title = "Backend Development"
	if err := s.UpsertNode(ctx, node); err != nil {
		t.Fatalf("update upsert: %v", err)
	}

	got, err := s.GetNode(ctx, "node-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Title != "Backend Development" {
		t.Errorf("title after update: got %q, want %q", got.Title, "Backend Development")
	}
}

func TestGetNode_NotFound(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	_, err := s.GetNode(ctx, "nonexistent")
	if err != domain.ErrNodeNotFound {
		t.Fatalf("expected ErrNodeNotFound, got %v", err)
	}
}

func TestDeleteNode_SoftDelete(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	node := domain.GraphNode{
		EngramID: "node-1",
		NodeType: domain.NodeTypeSubarea,
		Title:    "Go",
		Active:   true,
	}
	if err := s.UpsertNode(ctx, node); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	// Soft delete
	if err := s.DeleteNode(ctx, "node-1"); err != nil {
		t.Fatalf("delete: %v", err)
	}

	// Node should still exist but be inactive
	got, err := s.GetNode(ctx, "node-1")
	if err != nil {
		t.Fatalf("get after delete: %v", err)
	}
	if got.Active {
		t.Error("active after delete: got true, want false")
	}
}

func TestDeleteNode_NotFound(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	err := s.DeleteNode(ctx, "nonexistent")
	if err != domain.ErrNodeNotFound {
		t.Fatalf("expected ErrNodeNotFound, got %v", err)
	}
}

func TestDeleteNode_Idempotent(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	node := domain.GraphNode{
		EngramID: "node-1",
		NodeType: domain.NodeTypeDomain,
		Title:    "Test",
		Active:   true,
	}
	if err := s.UpsertNode(ctx, node); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	// First delete should succeed
	if err := s.DeleteNode(ctx, "node-1"); err != nil {
		t.Fatalf("first delete: %v", err)
	}

	// Second delete should return ErrNodeNotFound (already inactive)
	err := s.DeleteNode(ctx, "node-1")
	if err != domain.ErrNodeNotFound {
		t.Fatalf("expected ErrNodeNotFound on second delete, got %v", err)
	}
}

func TestListNodesByType(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	nodes := []domain.GraphNode{
		{EngramID: "d1", NodeType: domain.NodeTypeDomain, Title: "Backend", Active: true},
		{EngramID: "d2", NodeType: domain.NodeTypeDomain, Title: "Frontend", Active: true},
		{EngramID: "s1", NodeType: domain.NodeTypeSubarea, Title: "Go", Active: true},
		{EngramID: "p1", NodeType: domain.NodeTypeProject, Title: "Marrow", Active: true},
	}
	for _, n := range nodes {
		if err := s.UpsertNode(ctx, n); err != nil {
			t.Fatalf("upsert %s: %v", n.EngramID, err)
		}
	}

	domains, err := s.ListNodesByType(ctx, domain.NodeTypeDomain)
	if err != nil {
		t.Fatalf("list domains: %v", err)
	}
	if len(domains) != 2 {
		t.Fatalf("expected 2 domains, got %d", len(domains))
	}

	subareas, err := s.ListNodesByType(ctx, domain.NodeTypeSubarea)
	if err != nil {
		t.Fatalf("list subareas: %v", err)
	}
	if len(subareas) != 1 {
		t.Fatalf("expected 1 subarea, got %d", len(subareas))
	}
}

func TestAddEdge_And_GetEdges(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	if err := s.AddEdge(ctx, "d1", "s1", domain.EdgeContains); err != nil {
		t.Fatalf("add edge: %v", err)
	}

	edges, err := s.GetEdges(ctx, "d1", "from")
	if err != nil {
		t.Fatalf("get edges: %v", err)
	}
	if len(edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(edges))
	}
	if edges[0].FromID != "d1" {
		t.Errorf("from_id: got %q, want %q", edges[0].FromID, "d1")
	}
	if edges[0].ToID != "s1" {
		t.Errorf("to_id: got %q, want %q", edges[0].ToID, "s1")
	}
	if edges[0].Label != domain.EdgeContains {
		t.Errorf("label: got %q, want %q", edges[0].Label, domain.EdgeContains)
	}
}

func TestAddEdge_Duplicate(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	if err := s.AddEdge(ctx, "d1", "s1", domain.EdgeContains); err != nil {
		t.Fatalf("first add: %v", err)
	}

	err := s.AddEdge(ctx, "d1", "s1", domain.EdgeContains)
	if err != domain.ErrDuplicateNode {
		t.Fatalf("expected ErrDuplicateNode, got %v", err)
	}
}

func TestRemoveEdge(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	if err := s.AddEdge(ctx, "d1", "s1", domain.EdgeContains); err != nil {
		t.Fatalf("add edge: %v", err)
	}

	if err := s.RemoveEdge(ctx, "d1", "s1", domain.EdgeContains); err != nil {
		t.Fatalf("remove edge: %v", err)
	}

	edges, err := s.GetEdges(ctx, "d1", "from")
	if err != nil {
		t.Fatalf("get edges: %v", err)
	}
	if len(edges) != 0 {
		t.Errorf("expected 0 edges after removal, got %d", len(edges))
	}
}

func TestGetEdges_Directions(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	// d1 -> s1 (contains)
	// d2 -> s1 (contains)
	if err := s.AddEdge(ctx, "d1", "s1", domain.EdgeContains); err != nil {
		t.Fatalf("add edge 1: %v", err)
	}
	if err := s.AddEdge(ctx, "d2", "s1", domain.EdgeContains); err != nil {
		t.Fatalf("add edge 2: %v", err)
	}

	// "from" direction: edges where s1 is the source
	fromEdges, err := s.GetEdges(ctx, "s1", "from")
	if err != nil {
		t.Fatalf("from edges: %v", err)
	}
	if len(fromEdges) != 0 {
		t.Errorf("expected 0 'from' edges for s1, got %d", len(fromEdges))
	}

	// "to" direction: edges where s1 is the target
	toEdges, err := s.GetEdges(ctx, "s1", "to")
	if err != nil {
		t.Fatalf("to edges: %v", err)
	}
	if len(toEdges) != 2 {
		t.Errorf("expected 2 'to' edges for s1, got %d", len(toEdges))
	}

	// "both" direction
	bothEdges, err := s.GetEdges(ctx, "d1", "both")
	if err != nil {
		t.Fatalf("both edges: %v", err)
	}
	if len(bothEdges) != 1 {
		t.Errorf("expected 1 'both' edge for d1, got %d", len(bothEdges))
	}
}

func TestGetNeighbors(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	// Create nodes
	for _, n := range []domain.GraphNode{
		{EngramID: "d1", NodeType: domain.NodeTypeDomain, Title: "Backend", Active: true},
		{EngramID: "s1", NodeType: domain.NodeTypeSubarea, Title: "Go", Active: true},
		{EngramID: "s2", NodeType: domain.NodeTypeSubarea, Title: "Databases", Active: true},
	} {
		if err := s.UpsertNode(ctx, n); err != nil {
			t.Fatalf("upsert %s: %v", n.EngramID, err)
		}
	}

	// d1 -> s1, d1 -> s2 (contains)
	if err := s.AddEdge(ctx, "d1", "s1", domain.EdgeContains); err != nil {
		t.Fatalf("add edge 1: %v", err)
	}
	if err := s.AddEdge(ctx, "d1", "s2", domain.EdgeContains); err != nil {
		t.Fatalf("add edge 2: %v", err)
	}

	neighbors, err := s.GetNeighbors(ctx, "d1", domain.EdgeContains)
	if err != nil {
		t.Fatalf("get neighbors: %v", err)
	}
	if len(neighbors) != 2 {
		t.Fatalf("expected 2 neighbors, got %d", len(neighbors))
	}

	// Verify neighbor IDs
	ids := make(map[string]bool)
	for _, n := range neighbors {
		ids[n.EngramID] = true
	}
	if !ids["s1"] || !ids["s2"] {
		t.Errorf("expected neighbors s1 and s2, got %v", ids)
	}
}

func TestGetDomainTree(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	// Create domains
	for _, n := range []domain.GraphNode{
		{EngramID: "d1", NodeType: domain.NodeTypeDomain, Title: "Backend", Active: true},
		{EngramID: "d2", NodeType: domain.NodeTypeDomain, Title: "Frontend", Active: true},
		{EngramID: "s1", NodeType: domain.NodeTypeSubarea, Title: "Go", Active: true},
		{EngramID: "s2", NodeType: domain.NodeTypeSubarea, Title: "React", Active: true},
	} {
		if err := s.UpsertNode(ctx, n); err != nil {
			t.Fatalf("upsert %s: %v", n.EngramID, err)
		}
	}

	// d1 -> s1, d2 -> s2
	if err := s.AddEdge(ctx, "d1", "s1", domain.EdgeContains); err != nil {
		t.Fatalf("add edge 1: %v", err)
	}
	if err := s.AddEdge(ctx, "d2", "s2", domain.EdgeContains); err != nil {
		t.Fatalf("add edge 2: %v", err)
	}

	tree, err := s.GetDomainTree(ctx)
	if err != nil {
		t.Fatalf("get domain tree: %v", err)
	}
	if len(tree) != 2 {
		t.Fatalf("expected 2 domains in tree, got %d", len(tree))
	}

	// Find Backend domain
	var backend *domain.DomainWithSubareas
	for i := range tree {
		if tree[i].Domain.EngramID == "d1" {
			backend = &tree[i]
			break
		}
	}
	if backend == nil {
		t.Fatal("Backend domain not found in tree")
	}
	if len(backend.Subareas) != 1 {
		t.Fatalf("Backend should have 1 subarea, got %d", len(backend.Subareas))
	}
	if backend.Subareas[0].Title != "Go" {
		t.Errorf("subarea title: got %q, want %q", backend.Subareas[0].Title, "Go")
	}
}

func TestGetDomainTree_EmptyStore(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	tree, err := s.GetDomainTree(ctx)
	if err != nil {
		t.Fatalf("get domain tree: %v", err)
	}
	if len(tree) != 0 {
		t.Errorf("expected empty tree, got %d domains", len(tree))
	}
}

func TestListActiveProjects(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	// Create projects
	for _, n := range []domain.GraphNode{
		{EngramID: "p1", NodeType: domain.NodeTypeProject, Title: "Marrow", Active: true},
		{EngramID: "p2", NodeType: domain.NodeTypeProject, Title: "Old Project", Active: false},
		{EngramID: "s1", NodeType: domain.NodeTypeSubarea, Title: "Go", Active: true},
	} {
		if err := s.UpsertNode(ctx, n); err != nil {
			t.Fatalf("upsert %s: %v", n.EngramID, err)
		}
	}

	// Link p1 -> s1
	if err := s.AddEdge(ctx, "p1", "s1", domain.EdgeAppliesTo); err != nil {
		t.Fatalf("add edge: %v", err)
	}

	projects, err := s.ListActiveProjects(ctx)
	if err != nil {
		t.Fatalf("list active projects: %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("expected 1 active project, got %d", len(projects))
	}
	if projects[0].Name != "Marrow" {
		t.Errorf("name: got %q, want %q", projects[0].Name, "Marrow")
	}
	if !projects[0].Active {
		t.Error("active: got false, want true")
	}
	if len(projects[0].SubareaIDs) != 1 || projects[0].SubareaIDs[0] != "s1" {
		t.Errorf("subarea_ids: got %v, want [s1]", projects[0].SubareaIDs)
	}
}

func TestListActiveProjects_EmptyStore(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	projects, err := s.ListActiveProjects(ctx)
	if err != nil {
		t.Fatalf("list active projects: %v", err)
	}
	if len(projects) != 0 {
		t.Errorf("expected 0 projects, got %d", len(projects))
	}
}

func TestGetEdges_InvalidDirection(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	_, err := s.GetEdges(ctx, "node-1", "invalid")
	if err == nil {
		t.Fatal("expected error for invalid direction")
	}
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"Backend Development", "backend-development"},
		{"Go", "go"},
		{"  Spaces  ", "spaces"},
		{"Multiple   Spaces", "multiple-spaces"},
		{"Special!@#Chars", "specialchars"},
		{"UPPERCASE", "uppercase"},
		{"Mixed Case With-Hyphens", "mixed-case-with-hyphens"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := domain.Slugify(tt.name)
			if got != tt.want {
				t.Errorf("Slugify(%q) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

func TestNodeType_Valid(t *testing.T) {
	if !domain.NodeTypeDomain.Valid() {
		t.Error("NodeTypeDomain should be valid")
	}
	if !domain.NodeTypeSubarea.Valid() {
		t.Error("NodeTypeSubarea should be valid")
	}
	if !domain.NodeTypeProject.Valid() {
		t.Error("NodeTypeProject should be valid")
	}
	if domain.NodeType("invalid").Valid() {
		t.Error("invalid type should not be valid")
	}
}

// TestGraphStore_Interface verifies *Store satisfies domain.GraphStore.
func TestGraphStore_Interface(t *testing.T) {
	s := newTestStore(t)
	var _ domain.GraphStore = s
}
