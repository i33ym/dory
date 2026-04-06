package retrieve

import (
	"context"
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
