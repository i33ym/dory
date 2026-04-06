package dory

import (
	"context"
	"testing"
)

func TestHooks_Ingest(t *testing.T) {
	splitter := &fakeSplitter{
		chunks: []*Chunk{
			NewChunk("c1", "", "hello", nil),
			NewChunk("c2", "", "world", nil),
		},
	}

	var beforeDocCount int
	var afterChunkCount int
	var afterErr error

	hook := Hook{
		BeforeIngest: func(_ context.Context, docCount int) {
			beforeDocCount = docCount
		},
		AfterIngest: func(_ context.Context, chunkCount int, err error) {
			afterChunkCount = chunkCount
			afterErr = err
		},
	}

	p, err := NewPipeline(PipelineConfig{
		Splitter:  splitter,
		Embedder:  &fakeEmbedder{dim: 3},
		Store:     &fakeStore{},
		Retriever: &fakeRetriever{},
		Hooks:     []Hook{hook},
	})
	if err != nil {
		t.Fatalf("NewPipeline: %v", err)
	}

	doc, err := NewDocument("doc1", TextContent("hello world", ""))
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}

	ctx := context.Background()
	if err := p.Ingest(ctx, doc); err != nil {
		t.Fatalf("Ingest: %v", err)
	}

	if beforeDocCount != 1 {
		t.Errorf("BeforeIngest docCount = %d, want 1", beforeDocCount)
	}
	if afterChunkCount != 2 {
		t.Errorf("AfterIngest chunkCount = %d, want 2", afterChunkCount)
	}
	if afterErr != nil {
		t.Errorf("AfterIngest err = %v, want nil", afterErr)
	}
}

func TestHooks_Retrieve(t *testing.T) {
	units := []RetrievedUnit{
		NewChunk("c1", "doc1", "hello", nil),
		NewChunk("c2", "doc2", "world", nil),
	}

	var beforeQuery Query
	var afterResultCount int
	var afterErr error

	hook := Hook{
		BeforeRetrieve: func(_ context.Context, query Query) {
			beforeQuery = query
		},
		AfterRetrieve: func(_ context.Context, resultCount int, err error) {
			afterResultCount = resultCount
			afterErr = err
		},
	}

	p, err := NewPipeline(PipelineConfig{
		Splitter:  &fakeSplitter{},
		Embedder:  &fakeEmbedder{dim: 3},
		Store:     &fakeStore{},
		Retriever: &fakeRetriever{units: units},
		Hooks:     []Hook{hook},
	})
	if err != nil {
		t.Fatalf("NewPipeline: %v", err)
	}

	ctx := context.Background()
	q := Query{Text: "test query", TopK: 5}
	_, err = p.Retrieve(ctx, q)
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}

	if beforeQuery.Text != "test query" {
		t.Errorf("BeforeRetrieve query.Text = %q, want %q", beforeQuery.Text, "test query")
	}
	if beforeQuery.TopK != 5 {
		t.Errorf("BeforeRetrieve query.TopK = %d, want 5", beforeQuery.TopK)
	}
	if afterResultCount != 2 {
		t.Errorf("AfterRetrieve resultCount = %d, want 2", afterResultCount)
	}
	if afterErr != nil {
		t.Errorf("AfterRetrieve err = %v, want nil", afterErr)
	}
}

func TestHooks_Retrieve_WithReranker(t *testing.T) {
	units := []RetrievedUnit{
		NewChunk("c1", "doc1", "first", nil),
		NewChunk("c2", "doc2", "second", nil),
		NewChunk("c3", "doc3", "third", nil),
	}

	var beforeRerankQuery string
	var beforeRerankCount int
	var afterRerankCount int

	hook := Hook{
		BeforeRerank: func(_ context.Context, query string, candidateCount int) {
			beforeRerankQuery = query
			beforeRerankCount = candidateCount
		},
		AfterRerank: func(_ context.Context, resultCount int, err error) {
			afterRerankCount = resultCount
		},
	}

	p, err := NewPipeline(PipelineConfig{
		Splitter:  &fakeSplitter{},
		Embedder:  &fakeEmbedder{dim: 3},
		Store:     &fakeStore{},
		Retriever: &fakeRetriever{units: units},
		Reranker:  &fakeReranker{},
		Hooks:     []Hook{hook},
	})
	if err != nil {
		t.Fatalf("NewPipeline: %v", err)
	}

	ctx := context.Background()
	_, err = p.Retrieve(ctx, Query{Text: "rerank test", TopK: 5})
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}

	if beforeRerankQuery != "rerank test" {
		t.Errorf("BeforeRerank query = %q, want %q", beforeRerankQuery, "rerank test")
	}
	if beforeRerankCount != 3 {
		t.Errorf("BeforeRerank candidateCount = %d, want 3", beforeRerankCount)
	}
	if afterRerankCount != 3 {
		t.Errorf("AfterRerank resultCount = %d, want 3", afterRerankCount)
	}
}

func TestHooks_MultipleHooksCalledInOrder(t *testing.T) {
	splitter := &fakeSplitter{
		chunks: []*Chunk{
			NewChunk("c1", "", "hello", nil),
		},
	}

	var order []string

	hook1 := Hook{
		BeforeIngest: func(_ context.Context, _ int) {
			order = append(order, "hook1-before")
		},
		AfterIngest: func(_ context.Context, _ int, _ error) {
			order = append(order, "hook1-after")
		},
	}
	hook2 := Hook{
		BeforeIngest: func(_ context.Context, _ int) {
			order = append(order, "hook2-before")
		},
		AfterIngest: func(_ context.Context, _ int, _ error) {
			order = append(order, "hook2-after")
		},
	}

	p, err := NewPipeline(PipelineConfig{
		Splitter:  splitter,
		Embedder:  &fakeEmbedder{dim: 3},
		Store:     &fakeStore{},
		Retriever: &fakeRetriever{},
		Hooks:     []Hook{hook1, hook2},
	})
	if err != nil {
		t.Fatalf("NewPipeline: %v", err)
	}

	doc, err := NewDocument("doc1", TextContent("hello", ""))
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}

	ctx := context.Background()
	if err := p.Ingest(ctx, doc); err != nil {
		t.Fatalf("Ingest: %v", err)
	}

	expected := []string{"hook1-before", "hook2-before", "hook1-after", "hook2-after"}
	if len(order) != len(expected) {
		t.Fatalf("hook call count = %d, want %d", len(order), len(expected))
	}
	for i, v := range expected {
		if order[i] != v {
			t.Errorf("order[%d] = %q, want %q", i, order[i], v)
		}
	}
}

func TestHooks_NilFunctionsDontPanic(t *testing.T) {
	// Hook with all nil functions should not cause a panic.
	hook := Hook{}

	splitter := &fakeSplitter{
		chunks: []*Chunk{
			NewChunk("c1", "", "hello", nil),
		},
	}

	p, err := NewPipeline(PipelineConfig{
		Splitter:  splitter,
		Embedder:  &fakeEmbedder{dim: 3},
		Store:     &fakeStore{},
		Retriever: &fakeRetriever{units: []RetrievedUnit{NewChunk("c1", "doc1", "hello", nil)}},
		Reranker:  &fakeReranker{},
		Hooks:     []Hook{hook},
	})
	if err != nil {
		t.Fatalf("NewPipeline: %v", err)
	}

	doc, err := NewDocument("doc1", TextContent("hello", ""))
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}

	ctx := context.Background()

	// These should not panic.
	if err := p.Ingest(ctx, doc); err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	if _, err := p.Retrieve(ctx, Query{Text: "test", TopK: 5}); err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
}
