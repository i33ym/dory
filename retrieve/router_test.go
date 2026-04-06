package retrieve

import (
	"context"
	"strings"
	"testing"

	"github.com/i33ym/dory"
)

func TestRouter_MatchesFirstRoute(t *testing.T) {
	ctx := context.Background()

	c1 := dory.NewChunk("c1", "doc-1", "from route 1", nil).WithScore("test", 1.0)
	c2 := dory.NewChunk("c2", "doc-1", "from route 2", nil).WithScore("test", 1.0)

	router := NewRouter(RouterConfig{
		Routes: []Route{
			{
				Name:      "sql",
				Retriever: &fakeRetriever{results: []dory.RetrievedUnit{c1}},
				Match:     func(q dory.Query) bool { return strings.Contains(q.Text, "SELECT") },
			},
			{
				Name:      "default",
				Retriever: &fakeRetriever{results: []dory.RetrievedUnit{c2}},
				Match:     func(q dory.Query) bool { return true },
			},
		},
	})

	results, err := router.Retrieve(ctx, dory.Query{Text: "SELECT * FROM users", TopK: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].ID() != "c1" {
		t.Fatalf("expected route 1 result, got %v", results)
	}
}

func TestRouter_FallsThrough(t *testing.T) {
	ctx := context.Background()

	c2 := dory.NewChunk("c2", "doc-1", "from route 2", nil).WithScore("test", 1.0)

	router := NewRouter(RouterConfig{
		Routes: []Route{
			{
				Name:      "sql",
				Retriever: &fakeRetriever{},
				Match:     func(q dory.Query) bool { return strings.Contains(q.Text, "SELECT") },
			},
			{
				Name:      "default",
				Retriever: &fakeRetriever{results: []dory.RetrievedUnit{c2}},
				Match:     func(q dory.Query) bool { return true },
			},
		},
	})

	results, err := router.Retrieve(ctx, dory.Query{Text: "what is Go?", TopK: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].ID() != "c2" {
		t.Fatalf("expected default route result, got %v", results)
	}
}

func TestRouter_NoMatch(t *testing.T) {
	ctx := context.Background()

	router := NewRouter(RouterConfig{
		Routes: []Route{
			{
				Name:      "sql",
				Retriever: &fakeRetriever{},
				Match:     func(q dory.Query) bool { return false },
			},
		},
	})

	results, err := router.Retrieve(ctx, dory.Query{Text: "test", TopK: 5})
	if err != nil {
		t.Fatal(err)
	}
	if results != nil {
		t.Fatalf("expected nil results, got %v", results)
	}
}
