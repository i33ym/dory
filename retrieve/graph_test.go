package retrieve

import (
	"context"
	"testing"

	"github.com/i33ym/dory"
)

func TestGraph_Retrieve(t *testing.T) {
	ctx := context.Background()

	g := NewGraph()
	err := g.AddFacts(ctx,
		dory.NewGraphFact("f1", "doc-1", "Go", "is", "programming language", nil),
		dory.NewGraphFact("f2", "doc-1", "Python", "is", "programming language", nil),
		dory.NewGraphFact("f3", "doc-1", "Go", "has", "goroutines", nil),
	)
	if err != nil {
		t.Fatal(err)
	}

	results, err := g.Retrieve(ctx, dory.Query{Text: "Go programming", TopK: 10})
	if err != nil {
		t.Fatal(err)
	}

	if len(results) == 0 {
		t.Fatal("expected results, got none")
	}

	// The first result should be the one matching both terms.
	first := results[0]
	if first.Score() == 0 {
		t.Error("expected non-zero score")
	}
}

func TestGraph_Retrieve_NoMatch(t *testing.T) {
	ctx := context.Background()

	g := NewGraph()
	_ = g.AddFacts(ctx,
		dory.NewGraphFact("f1", "doc-1", "Go", "is", "programming language", nil),
	)

	results, err := g.Retrieve(ctx, dory.Query{Text: "Rust", TopK: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestGraph_Retrieve_TopK(t *testing.T) {
	ctx := context.Background()

	g := NewGraph()
	_ = g.AddFacts(ctx,
		dory.NewGraphFact("f1", "doc-1", "Go", "is", "fast", nil),
		dory.NewGraphFact("f2", "doc-1", "Go", "has", "channels", nil),
		dory.NewGraphFact("f3", "doc-1", "Go", "supports", "concurrency", nil),
	)

	results, err := g.Retrieve(ctx, dory.Query{Text: "Go", TopK: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestGraph_EmptyQuery(t *testing.T) {
	ctx := context.Background()

	g := NewGraph()
	_ = g.AddFacts(ctx,
		dory.NewGraphFact("f1", "doc-1", "Go", "is", "fast", nil),
	)

	results, err := g.Retrieve(ctx, dory.Query{Text: "", TopK: 10})
	if err != nil {
		t.Fatal(err)
	}
	if results != nil {
		t.Fatalf("expected nil results for empty query, got %d", len(results))
	}
}
