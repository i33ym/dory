package retrieve

import (
	"context"
	"testing"

	"github.com/i33ym/dory"
)

func TestEnsemble_Retrieve(t *testing.T) {
	ctx := context.Background()

	c1 := dory.NewChunk("c1", "doc-1", "alpha", nil).WithScore("test", 0.9)
	c2 := dory.NewChunk("c2", "doc-1", "beta", nil).WithScore("test", 0.8)
	c3 := dory.NewChunk("c3", "doc-1", "gamma", nil).WithScore("test", 0.7)

	r1 := &fakeRetriever{results: []dory.RetrievedUnit{c1, c2}}
	r2 := &fakeRetriever{results: []dory.RetrievedUnit{c3}}

	ensemble := NewEnsemble([]dory.Retriever{r1, r2})

	results, err := ensemble.Retrieve(ctx, dory.Query{Text: "test", TopK: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 3 {
		t.Fatalf("got %d results, want 3", len(results))
	}
	// Should be sorted by score descending.
	if results[0].ID() != "c1" {
		t.Errorf("first result ID = %s, want c1", results[0].ID())
	}
	if results[2].ID() != "c3" {
		t.Errorf("last result ID = %s, want c3", results[2].ID())
	}
}

func TestEnsemble_Deduplication(t *testing.T) {
	ctx := context.Background()

	c1a := dory.NewChunk("c1", "doc-1", "alpha", nil).WithScore("test", 0.9)
	c1b := dory.NewChunk("c1", "doc-1", "alpha", nil).WithScore("test", 0.5)
	c2 := dory.NewChunk("c2", "doc-1", "beta", nil).WithScore("test", 0.8)

	r1 := &fakeRetriever{results: []dory.RetrievedUnit{c1a}}
	r2 := &fakeRetriever{results: []dory.RetrievedUnit{c1b, c2}}

	ensemble := NewEnsemble([]dory.Retriever{r1, r2})

	results, err := ensemble.Retrieve(ctx, dory.Query{Text: "test", TopK: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2 (deduped)", len(results))
	}
}

func TestEnsemble_TopK(t *testing.T) {
	ctx := context.Background()

	c1 := dory.NewChunk("c1", "doc-1", "alpha", nil).WithScore("test", 0.9)
	c2 := dory.NewChunk("c2", "doc-1", "beta", nil).WithScore("test", 0.8)
	c3 := dory.NewChunk("c3", "doc-1", "gamma", nil).WithScore("test", 0.7)

	r1 := &fakeRetriever{results: []dory.RetrievedUnit{c1, c2, c3}}
	ensemble := NewEnsemble([]dory.Retriever{r1})

	results, err := ensemble.Retrieve(ctx, dory.Query{Text: "test", TopK: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
}
