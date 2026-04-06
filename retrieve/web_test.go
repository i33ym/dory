package retrieve

import (
	"context"
	"errors"
	"testing"

	"github.com/i33ym/dory"
)

func TestWeb_Retrieve(t *testing.T) {
	ctx := context.Background()

	w := NewWeb(WebConfig{
		SearchFunc: func(_ context.Context, query string, topK int) ([]WebResult, error) {
			return []WebResult{
				{URL: "https://example.com/1", Title: "Result 1", Snippet: "First result snippet"},
				{URL: "https://example.com/2", Title: "Result 2", Snippet: "Second result snippet"},
			}, nil
		},
	})

	results, err := w.Retrieve(ctx, dory.Query{Text: "Go tutorial", TopK: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}

	first := results[0]
	if first.ID() != "https://example.com/1" {
		t.Errorf("ID = %s, want URL", first.ID())
	}
	if first.SourceURI() != "https://example.com/1" {
		t.Errorf("SourceURI = %s, want URL", first.SourceURI())
	}
	if first.AsText() != "First result snippet" {
		t.Errorf("AsText = %s, want snippet", first.AsText())
	}
	meta := first.Metadata()
	if meta["title"] != "Result 1" {
		t.Errorf("metadata title = %v, want Result 1", meta["title"])
	}
}

func TestWeb_Retrieve_Error(t *testing.T) {
	ctx := context.Background()

	w := NewWeb(WebConfig{
		SearchFunc: func(_ context.Context, _ string, _ int) ([]WebResult, error) {
			return nil, errors.New("network error")
		},
	})

	_, err := w.Retrieve(ctx, dory.Query{Text: "test", TopK: 5})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestWeb_Retrieve_DefaultTopK(t *testing.T) {
	ctx := context.Background()

	var capturedTopK int
	w := NewWeb(WebConfig{
		SearchFunc: func(_ context.Context, _ string, topK int) ([]WebResult, error) {
			capturedTopK = topK
			return nil, nil
		},
	})

	_, err := w.Retrieve(ctx, dory.Query{Text: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if capturedTopK != 10 {
		t.Errorf("default topK = %d, want 10", capturedTopK)
	}
}
