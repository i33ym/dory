package retrieve

import (
	"context"
	"math"
	"testing"

	"github.com/i33ym/dory"
)

// fakeRetriever returns a fixed set of results.
type fakeRetriever struct {
	results []dory.RetrievedUnit
	err     error
}

func (f *fakeRetriever) Retrieve(_ context.Context, _ dory.Query) ([]dory.RetrievedUnit, error) {
	return f.results, f.err
}

func makeChunkUnit(id, text string, score float64) dory.RetrievedUnit {
	c := dory.NewChunk(id, "doc1", text, nil)
	return c.WithScore("test", score)
}

func TestHybrid_FusionTwoRetrievers(t *testing.T) {
	// Retriever A returns: a, b, c (ranked 1, 2, 3)
	// Retriever B returns: b, c, d (ranked 1, 2, 3)
	retA := &fakeRetriever{
		results: []dory.RetrievedUnit{
			makeChunkUnit("a", "chunk a", 0.9),
			makeChunkUnit("b", "chunk b", 0.8),
			makeChunkUnit("c", "chunk c", 0.7),
		},
	}
	retB := &fakeRetriever{
		results: []dory.RetrievedUnit{
			makeChunkUnit("b", "chunk b", 0.95),
			makeChunkUnit("c", "chunk c", 0.85),
			makeChunkUnit("d", "chunk d", 0.75),
		},
	}

	hybrid := NewHybrid([]dory.Retriever{retA, retB}, HybridConfig{K: 60})
	results, err := hybrid.Retrieve(context.Background(), dory.Query{Text: "query", TopK: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 4 {
		t.Fatalf("expected 4 deduplicated results, got %d", len(results))
	}

	// b appears in both at rank 1 (retA) and rank 0 (retB), so it should
	// have the highest RRF score: 1/(60+2) + 1/(60+1) = ~0.01613 + ~0.01639 = ~0.03252
	// a: 1/(60+1) = ~0.01639
	// c: 1/(60+3) + 1/(60+2) = ~0.01587 + ~0.01613 = ~0.03200
	// d: 1/(60+3) = ~0.01587
	if results[0].ID() != "b" {
		t.Errorf("expected 'b' to rank first (highest RRF), got %s", results[0].ID())
	}

	// Verify the score stage is "rrf_fusion".
	scores := results[0].Scores()
	lastScore := scores[len(scores)-1]
	if lastScore.Stage != "rrf_fusion" {
		t.Errorf("expected rrf_fusion stage, got %s", lastScore.Stage)
	}
}

func TestHybrid_Deduplication(t *testing.T) {
	// Both retrievers return the same single chunk.
	retA := &fakeRetriever{
		results: []dory.RetrievedUnit{makeChunkUnit("same", "same chunk", 0.9)},
	}
	retB := &fakeRetriever{
		results: []dory.RetrievedUnit{makeChunkUnit("same", "same chunk", 0.8)},
	}

	hybrid := NewHybrid([]dory.Retriever{retA, retB}, HybridConfig{K: 60})
	results, err := hybrid.Retrieve(context.Background(), dory.Query{Text: "query", TopK: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 deduplicated result, got %d", len(results))
	}

	// RRF score should be 1/(60+1) + 1/(60+1) = 2/(61)
	expectedScore := 2.0 / 61.0
	actualScore := results[0].Score()
	if math.Abs(actualScore-expectedScore) > 1e-9 {
		t.Errorf("expected RRF score %f, got %f", expectedScore, actualScore)
	}
}

func TestHybrid_RRFScoring(t *testing.T) {
	// Single retriever, verify exact RRF scores.
	ret := &fakeRetriever{
		results: []dory.RetrievedUnit{
			makeChunkUnit("first", "first", 0.9),
			makeChunkUnit("second", "second", 0.8),
		},
	}

	k := 60
	hybrid := NewHybrid([]dory.Retriever{ret}, HybridConfig{K: k})
	results, err := hybrid.Retrieve(context.Background(), dory.Query{Text: "query", TopK: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	expected1 := 1.0 / float64(k+1)
	expected2 := 1.0 / float64(k+2)

	if math.Abs(results[0].Score()-expected1) > 1e-9 {
		t.Errorf("first result: expected score %f, got %f", expected1, results[0].Score())
	}
	if math.Abs(results[1].Score()-expected2) > 1e-9 {
		t.Errorf("second result: expected score %f, got %f", expected2, results[1].Score())
	}
}

func TestHybrid_TopK(t *testing.T) {
	ret := &fakeRetriever{
		results: []dory.RetrievedUnit{
			makeChunkUnit("a", "a", 0.9),
			makeChunkUnit("b", "b", 0.8),
			makeChunkUnit("c", "c", 0.7),
		},
	}

	hybrid := NewHybrid([]dory.Retriever{ret}, HybridConfig{K: 60})
	results, err := hybrid.Retrieve(context.Background(), dory.Query{Text: "query", TopK: 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results with TopK=2, got %d", len(results))
	}
}
