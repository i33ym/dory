package store

import (
	"context"
	"fmt"
	"math/rand"
	"testing"

	"github.com/i33ym/dory"
)

func makeChunk(id string, vec []float32, meta map[string]any) *dory.Chunk {
	c := dory.NewChunk(id, "doc-1", "text for "+id, meta)
	c.Vector = vec
	return c
}

func TestMemory_Store(t *testing.T) {
	ctx := context.Background()
	m := NewMemory()

	chunks := []*dory.Chunk{
		makeChunk("c1", []float32{1, 0, 0}, nil),
		makeChunk("c2", []float32{0, 1, 0}, nil),
	}

	if err := m.Store(ctx, chunks); err != nil {
		t.Fatal(err)
	}

	// Storing again with same ID overwrites.
	updated := makeChunk("c1", []float32{0, 0, 1}, nil)
	if err := m.Store(ctx, []*dory.Chunk{updated}); err != nil {
		t.Fatal(err)
	}

	results, err := m.Search(ctx, dory.SearchRequest{
		QueryVector: []float32{0, 0, 1},
		TopK:        10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
	// Updated c1 should be the best match for {0,0,1}.
	if results[0].Chunk.ID() != "c1" {
		t.Errorf("got top result %q, want %q", results[0].Chunk.ID(), "c1")
	}
}

func TestMemory_Search_TopK(t *testing.T) {
	ctx := context.Background()
	m := NewMemory()

	chunks := []*dory.Chunk{
		makeChunk("c1", []float32{1, 0, 0}, nil),
		makeChunk("c2", []float32{0.9, 0.1, 0}, nil),
		makeChunk("c3", []float32{0, 1, 0}, nil),
	}
	if err := m.Store(ctx, chunks); err != nil {
		t.Fatal(err)
	}

	results, err := m.Search(ctx, dory.SearchRequest{
		QueryVector: []float32{1, 0, 0},
		TopK:        2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
}

func TestMemory_Search_OrderedByScore(t *testing.T) {
	ctx := context.Background()
	m := NewMemory()

	chunks := []*dory.Chunk{
		makeChunk("far", []float32{0, 1, 0}, nil),
		makeChunk("close", []float32{0.95, 0.05, 0}, nil),
		makeChunk("exact", []float32{1, 0, 0}, nil),
	}
	if err := m.Store(ctx, chunks); err != nil {
		t.Fatal(err)
	}

	results, err := m.Search(ctx, dory.SearchRequest{
		QueryVector: []float32{1, 0, 0},
		TopK:        3,
	})
	if err != nil {
		t.Fatal(err)
	}

	if results[0].Chunk.ID() != "exact" {
		t.Errorf("expected 'exact' first, got %q", results[0].Chunk.ID())
	}
	if results[1].Chunk.ID() != "close" {
		t.Errorf("expected 'close' second, got %q", results[1].Chunk.ID())
	}
	if results[2].Chunk.ID() != "far" {
		t.Errorf("expected 'far' third, got %q", results[2].Chunk.ID())
	}
}

func TestMemory_Search_WithFilter(t *testing.T) {
	ctx := context.Background()
	m := NewMemory()

	chunks := []*dory.Chunk{
		makeChunk("c1", []float32{1, 0, 0}, map[string]any{"tenant": "acme"}),
		makeChunk("c2", []float32{0.9, 0.1, 0}, map[string]any{"tenant": "globex"}),
		makeChunk("c3", []float32{0.8, 0.2, 0}, map[string]any{"tenant": "acme"}),
	}
	if err := m.Store(ctx, chunks); err != nil {
		t.Fatal(err)
	}

	results, err := m.Search(ctx, dory.SearchRequest{
		QueryVector: []float32{1, 0, 0},
		TopK:        10,
		Filter:      &dory.MetadataFilter{Field: "tenant", Op: dory.FilterOpEq, Value: "acme"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
	for _, r := range results {
		if r.Chunk.Metadata()["tenant"] != "acme" {
			t.Errorf("got tenant %v, want acme", r.Chunk.Metadata()["tenant"])
		}
	}
}

func TestMemory_Search_SkipsNilVectors(t *testing.T) {
	ctx := context.Background()
	m := NewMemory()

	chunks := []*dory.Chunk{
		makeChunk("has-vec", []float32{1, 0, 0}, nil),
		dory.NewChunk("no-vec", "doc-1", "no vector", nil), // no Vector set
	}
	if err := m.Store(ctx, chunks); err != nil {
		t.Fatal(err)
	}

	results, err := m.Search(ctx, dory.SearchRequest{
		QueryVector: []float32{1, 0, 0},
		TopK:        10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
}

func TestMemory_Search_EmptyStore(t *testing.T) {
	ctx := context.Background()
	m := NewMemory()

	results, err := m.Search(ctx, dory.SearchRequest{
		QueryVector: []float32{1, 0, 0},
		TopK:        5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Fatalf("got %d results, want 0", len(results))
	}
}

func TestMemory_Delete(t *testing.T) {
	ctx := context.Background()
	m := NewMemory()

	chunks := []*dory.Chunk{
		makeChunk("c1", []float32{1, 0, 0}, nil),
		makeChunk("c2", []float32{0, 1, 0}, nil),
		makeChunk("c3", []float32{0, 0, 1}, nil),
	}
	if err := m.Store(ctx, chunks); err != nil {
		t.Fatal(err)
	}

	if err := m.Delete(ctx, []string{"c1", "c3"}); err != nil {
		t.Fatal(err)
	}

	results, err := m.Search(ctx, dory.SearchRequest{
		QueryVector: []float32{1, 1, 1},
		TopK:        10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].Chunk.ID() != "c2" {
		t.Errorf("got %q, want %q", results[0].Chunk.ID(), "c2")
	}
}

func TestMemory_Delete_NonExistent(t *testing.T) {
	ctx := context.Background()
	m := NewMemory()

	// Deleting IDs that don't exist should not error.
	if err := m.Delete(ctx, []string{"nope"}); err != nil {
		t.Fatal(err)
	}
}

func BenchmarkMemory_Search(b *testing.B) {
	const (
		numChunks = 1000
		dims      = 1536
	)

	rng := rand.New(rand.NewSource(42))
	ctx := context.Background()
	m := NewMemory()

	chunks := make([]*dory.Chunk, numChunks)
	for i := range chunks {
		vec := make([]float32, dims)
		for j := range vec {
			vec[j] = rng.Float32()
		}
		chunks[i] = makeChunk(fmt.Sprintf("c%d", i), vec, nil)
	}
	if err := m.Store(ctx, chunks); err != nil {
		b.Fatal(err)
	}

	queryVec := make([]float32, dims)
	for i := range queryVec {
		queryVec[i] = rng.Float32()
	}
	req := dory.SearchRequest{QueryVector: queryVec, TopK: 10}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = m.Search(ctx, req)
	}
}
