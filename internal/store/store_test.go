package store_test

import (
	"context"
	"os"
	"testing"

	"github.com/Nick-2455/marrow/internal/domain"
	"github.com/Nick-2455/marrow/internal/store"
)

func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	// Use temp file instead of :memory: because WAL mode doesn't work
	// properly with in-memory SQLite databases.
	tmpFile, err := os.CreateTemp("", "marrow-test-*.db")
	if err != nil {
		t.Fatalf("create temp db: %v", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()

	s, err := store.Open(tmpPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := s.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() {
		_ = s.Close()
		_ = os.Remove(tmpPath)
	})
	return s
}

func TestTriagePosition_CRUD(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	// Get on empty store should return ErrTriageNotFound
	_, err := s.GetTriagePosition(ctx, "res-1")
	if err != domain.ErrTriageNotFound {
		t.Fatalf("expected ErrTriageNotFound, got %v", err)
	}

	// Set a position
	pos := domain.TriagePosition{
		ResourceID: "res-1",
		Bucket:     domain.BucketInbox,
	}
	if err := s.SetTriagePosition(ctx, pos); err != nil {
		t.Fatalf("set: %v", err)
	}

	// Get should return it
	got, err := s.GetTriagePosition(ctx, "res-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ResourceID != "res-1" {
		t.Errorf("resource_id: got %q, want %q", got.ResourceID, "res-1")
	}
	if got.Bucket != domain.BucketInbox {
		t.Errorf("bucket: got %q, want %q", got.Bucket, domain.BucketInbox)
	}

	// Update the position
	pos.Bucket = domain.BucketActive
	if err := s.SetTriagePosition(ctx, pos); err != nil {
		t.Fatalf("update: %v", err)
	}

	got, err = s.GetTriagePosition(ctx, "res-1")
	if err != nil {
		t.Fatalf("get after update: %v", err)
	}
	if got.Bucket != domain.BucketActive {
		t.Errorf("bucket after update: got %q, want %q", got.Bucket, domain.BucketActive)
	}
}

func TestGetAllTriagePositions(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	// Empty store
	all, err := s.GetAllTriagePositions(ctx)
	if err != nil {
		t.Fatalf("get all: %v", err)
	}
	if len(all) != 0 {
		t.Errorf("expected 0 positions, got %d", len(all))
	}

	// Add positions
	positions := []domain.TriagePosition{
		{ResourceID: "res-1", Bucket: domain.BucketInbox},
		{ResourceID: "res-2", Bucket: domain.BucketActive},
		{ResourceID: "res-3", Bucket: domain.BucketLater},
	}
	for _, p := range positions {
		if err := s.SetTriagePosition(ctx, p); err != nil {
			t.Fatalf("set %s: %v", p.ResourceID, err)
		}
	}

	all, err = s.GetAllTriagePositions(ctx)
	if err != nil {
		t.Fatalf("get all: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3 positions, got %d", len(all))
	}
}

func TestSearchCache_Miss(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	_, found, err := s.GetCachedSearch(ctx, "nonexistent query")
	if err != nil {
		t.Fatalf("get cached: %v", err)
	}
	if found {
		t.Error("expected cache miss")
	}
}

func TestSearchCache_CRUD(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	resources := []domain.Resource{
		{ID: "res-1", Title: "Test Resource", Bucket: domain.BucketInbox},
		{ID: "res-2", Title: "Another Resource", Bucket: domain.BucketActive},
	}

	// Cache a search
	if err := s.CacheSearch(ctx, "test query", resources); err != nil {
		t.Fatalf("cache: %v", err)
	}

	// Retrieve it
	got, found, err := s.GetCachedSearch(ctx, "test query")
	if err != nil {
		t.Fatalf("get cached: %v", err)
	}
	if !found {
		t.Fatal("expected cache hit")
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 results, got %d", len(got))
	}
	if got[0].Title != "Test Resource" {
		t.Errorf("title[0]: got %q, want %q", got[0].Title, "Test Resource")
	}
}

func TestSearchCache_Overwrite(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	// Cache first query
	first := []domain.Resource{{ID: "res-1", Title: "First"}}
	if err := s.CacheSearch(ctx, "query", first); err != nil {
		t.Fatal(err)
	}

	// Overwrite with different results
	second := []domain.Resource{{ID: "res-2", Title: "Second"}, {ID: "res-3", Title: "Third"}}
	if err := s.CacheSearch(ctx, "query", second); err != nil {
		t.Fatal(err)
	}

	got, found, err := s.GetCachedSearch(ctx, "query")
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("expected cache hit")
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 results after overwrite, got %d", len(got))
	}
}

func TestEvictExpiredCache(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	resources := []domain.Resource{{ID: "res-1", Title: "Test"}}
	if err := s.CacheSearch(ctx, "test", resources); err != nil {
		t.Fatal(err)
	}

	// Should be cached
	_, found, err := s.GetCachedSearch(ctx, "test")
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("expected cache hit before eviction")
	}

	// Evict (nothing expired yet, but should not error)
	if err := s.EvictExpiredCache(); err != nil {
		t.Fatalf("evict: %v", err)
	}

	// Should still be there
	_, found, err = s.GetCachedSearch(ctx, "test")
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("expected cache hit after evict (nothing expired)")
	}
}
