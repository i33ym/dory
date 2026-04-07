package retrieve

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"github.com/i33ym/dory"
)

func TestBM25_EmptyIndex(t *testing.T) {
	bm := NewBM25(BM25Config{})
	results, err := bm.Retrieve(context.Background(), dory.Query{Text: "hello", TopK: 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results from empty index, got %d", len(results))
	}
}

func TestBM25_IndexAndRetrieve(t *testing.T) {
	bm := NewBM25(BM25Config{})

	chunks := []*dory.Chunk{
		dory.NewChunk("1", "doc1", "the cat sat on the mat", nil),
		dory.NewChunk("2", "doc1", "the dog ran in the park", nil),
		dory.NewChunk("3", "doc2", "a quick brown fox jumps over the lazy dog", nil),
	}

	if err := bm.Index(context.Background(), chunks); err != nil {
		t.Fatalf("index error: %v", err)
	}

	results, err := bm.Retrieve(context.Background(), dory.Query{Text: "cat mat", TopK: 3})
	if err != nil {
		t.Fatalf("retrieve error: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected at least one result")
	}

	// The chunk about "cat" and "mat" should be ranked first.
	if results[0].ID() != "1" {
		t.Errorf("expected chunk 1 to rank first, got %s", results[0].ID())
	}

	// Verify score is positive.
	if results[0].Score() <= 0 {
		t.Errorf("expected positive score, got %f", results[0].Score())
	}

	// Check score stage is "bm25".
	scores := results[0].Scores()
	if len(scores) == 0 || scores[0].Stage != "bm25" {
		t.Errorf("expected bm25 score stage, got %v", scores)
	}
}

func TestBM25_KeywordMatchScoresHigher(t *testing.T) {
	bm := NewBM25(BM25Config{})

	chunks := []*dory.Chunk{
		dory.NewChunk("relevant", "doc1", "golang concurrency goroutines channels", nil),
		dory.NewChunk("unrelated", "doc1", "chocolate cake recipe baking oven", nil),
	}

	if err := bm.Index(context.Background(), chunks); err != nil {
		t.Fatalf("index error: %v", err)
	}

	results, err := bm.Retrieve(context.Background(), dory.Query{Text: "golang goroutines", TopK: 5})
	if err != nil {
		t.Fatalf("retrieve error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result (only the matching chunk), got %d", len(results))
	}

	if results[0].ID() != "relevant" {
		t.Errorf("expected 'relevant' chunk, got %s", results[0].ID())
	}
}

func TestBM25_TopK(t *testing.T) {
	bm := NewBM25(BM25Config{})

	chunks := []*dory.Chunk{
		dory.NewChunk("1", "doc1", "apple banana cherry", nil),
		dory.NewChunk("2", "doc1", "apple date elderberry", nil),
		dory.NewChunk("3", "doc1", "apple fig grape", nil),
	}

	if err := bm.Index(context.Background(), chunks); err != nil {
		t.Fatalf("index error: %v", err)
	}

	results, err := bm.Retrieve(context.Background(), dory.Query{Text: "apple", TopK: 2})
	if err != nil {
		t.Fatalf("retrieve error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results with TopK=2, got %d", len(results))
	}
}

func TestBM25_MultipleIndexCalls(t *testing.T) {
	bm := NewBM25(BM25Config{})

	if err := bm.Index(context.Background(), []*dory.Chunk{
		dory.NewChunk("1", "doc1", "machine learning neural networks", nil),
	}); err != nil {
		t.Fatalf("first index error: %v", err)
	}

	if err := bm.Index(context.Background(), []*dory.Chunk{
		dory.NewChunk("2", "doc1", "deep learning transformers attention", nil),
	}); err != nil {
		t.Fatalf("second index error: %v", err)
	}

	results, err := bm.Retrieve(context.Background(), dory.Query{Text: "learning", TopK: 10})
	if err != nil {
		t.Fatalf("retrieve error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results across both index calls, got %d", len(results))
	}
}

func TestBM25_RespectsFilters(t *testing.T) {
	bm := NewBM25(BM25Config{})

	chunks := []*dory.Chunk{
		dory.NewChunk("c1", "doc1", "dory retrieval library", map[string]any{"tenant_id": "acme"}),
		dory.NewChunk("c2", "doc2", "dory retrieval pipeline", map[string]any{"tenant_id": "acme"}),
		dory.NewChunk("c3", "doc3", "dory retrieval search", map[string]any{"tenant_id": "globex"}),
	}

	if err := bm.Index(context.Background(), chunks); err != nil {
		t.Fatal(err)
	}

	// Without filter: all 3 match "dory retrieval".
	all, err := bm.Retrieve(context.Background(), dory.Query{Text: "dory retrieval", TopK: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 {
		t.Fatalf("unfiltered: got %d results, want 3", len(all))
	}

	// With tenant filter: only acme chunks.
	filtered, err := bm.Retrieve(context.Background(), dory.Query{
		Text: "dory retrieval",
		TopK: 10,
		Filters: []dory.MetadataFilter{
			{Field: "tenant_id", Op: dory.FilterOpEq, Value: "acme"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered) != 2 {
		t.Fatalf("filtered: got %d results, want 2", len(filtered))
	}
	for _, r := range filtered {
		if r.Metadata()["tenant_id"] != "acme" {
			t.Errorf("got tenant %v, want acme", r.Metadata()["tenant_id"])
		}
	}
}

func TestBM25_NilMetadataSkippedByFilter(t *testing.T) {
	bm := NewBM25(BM25Config{})

	chunks := []*dory.Chunk{
		dory.NewChunk("c1", "doc1", "dory search", nil),                                 // no metadata
		dory.NewChunk("c2", "doc2", "dory search", map[string]any{"tenant_id": "acme"}), // matches
	}

	if err := bm.Index(context.Background(), chunks); err != nil {
		t.Fatal(err)
	}

	results, err := bm.Retrieve(context.Background(), dory.Query{
		Text: "dory search",
		TopK: 10,
		Filters: []dory.MetadataFilter{
			{Field: "tenant_id", Op: dory.FilterOpEq, Value: "acme"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].ID() != "c2" {
		t.Errorf("got %q, want c2", results[0].ID())
	}
}

func BenchmarkBM25_Retrieve(b *testing.B) {
	const numChunks = 1000

	// Vocabulary to build realistic chunk text from.
	vocab := []string{
		"machine", "learning", "neural", "network", "deep",
		"transformer", "attention", "embedding", "vector", "search",
		"retrieval", "augmented", "generation", "language", "model",
		"training", "inference", "token", "context", "window",
		"gradient", "descent", "optimization", "loss", "function",
		"batch", "epoch", "layer", "activation", "weight",
	}

	rng := rand.New(rand.NewSource(42))
	bm := NewBM25(BM25Config{})

	chunks := make([]*dory.Chunk, numChunks)
	for i := range chunks {
		// Build a chunk with 20-40 words from vocab.
		wordCount := 20 + rng.Intn(21)
		words := make([]string, wordCount)
		for j := range words {
			words[j] = vocab[rng.Intn(len(vocab))]
		}
		chunks[i] = dory.NewChunk(fmt.Sprintf("c%d", i), "doc1", strings.Join(words, " "), nil)
	}

	if err := bm.Index(context.Background(), chunks); err != nil {
		b.Fatal(err)
	}

	q := dory.Query{Text: "deep learning transformer attention model", TopK: 10}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = bm.Retrieve(context.Background(), q)
	}
}
