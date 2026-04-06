package retrieve

import (
	"context"
	"testing"

	"github.com/i33ym/dory"
)

// stubRetriever returns a fixed set of results.
type stubRetriever struct {
	results []dory.RetrievedUnit
}

func (f *stubRetriever) Retrieve(_ context.Context, _ dory.Query) ([]dory.RetrievedUnit, error) {
	return f.results, nil
}

func TestSmallToBig_ExpandsToParent(t *testing.T) {
	parent := dory.NewChunk("parent-1", "doc-1", "This is the full parent paragraph with lots of context.", nil)

	child := dory.NewChunk("child-1", "doc-1", "parent paragraph", nil)
	child.ParentID = "parent-1"
	scoredChild := child.WithScore("vector", 0.95)

	retriever := &stubRetriever{results: []dory.RetrievedUnit{scoredChild}}
	stb := NewSmallToBig(SmallToBigConfig{
		Retriever: retriever,
		Parents:   map[string]*dory.Chunk{"parent-1": parent},
	})

	results, err := stb.Retrieve(context.Background(), dory.Query{Text: "test", TopK: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}

	// Result should be the parent chunk text.
	if results[0].AsText() != parent.AsText() {
		t.Errorf("got text %q, want %q", results[0].AsText(), parent.AsText())
	}

	// Score should carry over from the child.
	if results[0].Score() != 0.95 {
		t.Errorf("got score %f, want 0.95", results[0].Score())
	}

	// Score stage should be "small_to_big".
	scores := results[0].Scores()
	if len(scores) != 1 || scores[0].Stage != "small_to_big" {
		t.Errorf("expected small_to_big score stage, got %v", scores)
	}
}

func TestSmallToBig_DeduplicatesByParent(t *testing.T) {
	parent := dory.NewChunk("parent-1", "doc-1", "Full parent text.", nil)

	child1 := dory.NewChunk("child-1", "doc-1", "text A", nil)
	child1.ParentID = "parent-1"
	scored1 := child1.WithScore("vector", 0.8)

	child2 := dory.NewChunk("child-2", "doc-1", "text B", nil)
	child2.ParentID = "parent-1"
	scored2 := child2.WithScore("vector", 0.9)

	retriever := &stubRetriever{results: []dory.RetrievedUnit{scored1, scored2}}
	stb := NewSmallToBig(SmallToBigConfig{
		Retriever: retriever,
		Parents:   map[string]*dory.Chunk{"parent-1": parent},
	})

	results, err := stb.Retrieve(context.Background(), dory.Query{Text: "test", TopK: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1 (deduplicated)", len(results))
	}

	// Should keep the higher score (0.9 from child2).
	if results[0].Score() != 0.9 {
		t.Errorf("got score %f, want 0.9 (highest child score)", results[0].Score())
	}
}

func TestSmallToBig_PassesThroughNonChunks(t *testing.T) {
	fact := dory.NewGraphFact("f1", "doc-1", "Go", "is", "language", nil)
	scored := fact.WithScore("vector", 0.7)

	retriever := &stubRetriever{results: []dory.RetrievedUnit{scored}}
	stb := NewSmallToBig(SmallToBigConfig{
		Retriever: retriever,
		Parents:   map[string]*dory.Chunk{},
	})

	results, err := stb.Retrieve(context.Background(), dory.Query{Text: "test", TopK: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].ID() != "f1" {
		t.Errorf("got ID %q, want %q", results[0].ID(), "f1")
	}
}

func TestSmallToBig_NoParentID(t *testing.T) {
	child := dory.NewChunk("child-1", "doc-1", "orphan chunk", nil)
	scored := child.WithScore("vector", 0.85)

	retriever := &stubRetriever{results: []dory.RetrievedUnit{scored}}
	stb := NewSmallToBig(SmallToBigConfig{
		Retriever: retriever,
		Parents:   map[string]*dory.Chunk{},
	})

	results, err := stb.Retrieve(context.Background(), dory.Query{Text: "test", TopK: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	// Should pass through the child as-is.
	if results[0].AsText() != "orphan chunk" {
		t.Errorf("got text %q, want %q", results[0].AsText(), "orphan chunk")
	}
}

func TestSmallToBig_ParentNotInMap(t *testing.T) {
	child := dory.NewChunk("child-1", "doc-1", "child text", nil)
	child.ParentID = "missing-parent"
	scored := child.WithScore("vector", 0.75)

	retriever := &stubRetriever{results: []dory.RetrievedUnit{scored}}
	stb := NewSmallToBig(SmallToBigConfig{
		Retriever: retriever,
		Parents:   map[string]*dory.Chunk{},
	})

	results, err := stb.Retrieve(context.Background(), dory.Query{Text: "test", TopK: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	// Should pass through the child since parent is missing.
	if results[0].AsText() != "child text" {
		t.Errorf("got text %q, want %q", results[0].AsText(), "child text")
	}
}

func TestSmallToBig_MixedResults(t *testing.T) {
	parent1 := dory.NewChunk("parent-1", "doc-1", "Parent one text.", nil)
	parent2 := dory.NewChunk("parent-2", "doc-1", "Parent two text.", nil)

	child1 := dory.NewChunk("child-1", "doc-1", "c1", nil)
	child1.ParentID = "parent-1"
	child2 := dory.NewChunk("child-2", "doc-1", "c2", nil)
	child2.ParentID = "parent-2"
	child3 := dory.NewChunk("child-3", "doc-1", "c3", nil)
	child3.ParentID = "parent-1" // same parent as child1

	retriever := &stubRetriever{results: []dory.RetrievedUnit{
		child1.WithScore("vector", 0.9),
		child2.WithScore("vector", 0.85),
		child3.WithScore("vector", 0.95), // higher score for same parent
	}}

	stb := NewSmallToBig(SmallToBigConfig{
		Retriever: retriever,
		Parents: map[string]*dory.Chunk{
			"parent-1": parent1,
			"parent-2": parent2,
		},
	})

	results, err := stb.Retrieve(context.Background(), dory.Query{Text: "test", TopK: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2 (two unique parents)", len(results))
	}

	// First result is parent-1 with updated score from child3 (0.95 > 0.9).
	if results[0].Score() != 0.95 {
		t.Errorf("parent-1 score: got %f, want 0.95", results[0].Score())
	}
	if results[1].Score() != 0.85 {
		t.Errorf("parent-2 score: got %f, want 0.85", results[1].Score())
	}
}
