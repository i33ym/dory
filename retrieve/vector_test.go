package retrieve

import (
	"context"
	"testing"

	"github.com/i33ym/dory"
)

// fakeEmbedder returns a fixed vector for any input.
type fakeEmbedder struct {
	vec  []float32
	dims int
}

func (f *fakeEmbedder) Embed(_ context.Context, _ string) ([]float32, error) {
	return f.vec, nil
}

func (f *fakeEmbedder) EmbedBatch(_ context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i := range texts {
		out[i] = f.vec
	}
	return out, nil
}

func (f *fakeEmbedder) Dimensions() int { return f.dims }

// fakeStore is a minimal VectorStore for testing the retriever.
type fakeStore struct {
	chunks []*dory.Chunk
}

func (f *fakeStore) Store(_ context.Context, chunks []*dory.Chunk) error {
	f.chunks = append(f.chunks, chunks...)
	return nil
}

func (f *fakeStore) Search(_ context.Context, req dory.SearchRequest) ([]dory.ScoredChunk, error) {
	topK := min(req.TopK, len(f.chunks))
	out := make([]dory.ScoredChunk, topK)
	for i := 0; i < topK; i++ {
		out[i] = dory.ScoredChunk{Chunk: f.chunks[i], Score: 1.0 - float64(i)*0.1}
	}
	return out, nil
}

func (f *fakeStore) Delete(_ context.Context, _ []string) error { return nil }

func TestVector_Retrieve(t *testing.T) {
	ctx := context.Background()

	c1 := dory.NewChunk("c1", "doc-1", "first chunk", nil)
	c1.Vector = []float32{1, 0, 0}
	c2 := dory.NewChunk("c2", "doc-1", "second chunk", nil)
	c2.Vector = []float32{0, 1, 0}

	store := &fakeStore{chunks: []*dory.Chunk{c1, c2}}
	embedder := &fakeEmbedder{vec: []float32{1, 0, 0}, dims: 3}
	retriever := NewVector(store, embedder)

	results, err := retriever.Retrieve(ctx, dory.Query{Text: "test", TopK: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
	if results[0].Score() != 1.0 {
		t.Errorf("got score %f, want 1.0", results[0].Score())
	}
	if results[1].Score() != 0.9 {
		t.Errorf("got score %f, want 0.9", results[1].Score())
	}
}

func TestVector_Retrieve_DefaultTopK(t *testing.T) {
	ctx := context.Background()

	chunks := make([]*dory.Chunk, 15)
	for i := range chunks {
		c := dory.NewChunk("c"+string(rune('a'+i)), "doc-1", "text", nil)
		c.Vector = []float32{1, 0, 0}
		chunks[i] = c
	}

	store := &fakeStore{chunks: chunks}
	embedder := &fakeEmbedder{vec: []float32{1, 0, 0}, dims: 3}
	retriever := NewVector(store, embedder)

	// TopK=0 should default to 10.
	results, err := retriever.Retrieve(ctx, dory.Query{Text: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 10 {
		t.Fatalf("got %d results, want 10 (default)", len(results))
	}
}

func TestVector_Retrieve_WithFilter(t *testing.T) {
	ctx := context.Background()

	store := &fakeStore{}
	embedder := &fakeEmbedder{vec: []float32{1, 0, 0}, dims: 3}
	retriever := NewVector(store, embedder)

	_, err := retriever.Retrieve(ctx, dory.Query{
		Text: "test",
		TopK: 5,
		Filters: []dory.MetadataFilter{
			{Field: "tenant", Op: dory.FilterOpEq, Value: "acme"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}
