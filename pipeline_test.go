package dory

import (
	"context"
	"testing"
)

// --- Fake implementations for testing ---

type fakeSplitter struct {
	chunks []*Chunk
}

func (f *fakeSplitter) Split(_ context.Context, doc *Document) ([]*Chunk, error) {
	// Return pre-configured chunks, setting their sourceDocID to doc's ID.
	out := make([]*Chunk, len(f.chunks))
	for i, c := range f.chunks {
		out[i] = NewChunk(c.ID(), doc.ID(), c.Text(), c.Metadata())
	}
	return out, nil
}

type fakeEmbedder struct {
	dim        int
	embedCalls int
	batchCalls int
}

func (f *fakeEmbedder) Embed(_ context.Context, _ string) ([]float32, error) {
	f.embedCalls++
	return make([]float32, f.dim), nil
}

func (f *fakeEmbedder) EmbedBatch(_ context.Context, texts []string) ([][]float32, error) {
	f.batchCalls++
	vecs := make([][]float32, len(texts))
	for i := range texts {
		vecs[i] = make([]float32, f.dim)
		// Put a distinguishing value so we can verify vectors are set.
		if f.dim > 0 {
			vecs[i][0] = float32(i + 1)
		}
	}
	return vecs, nil
}

func (f *fakeEmbedder) Dimensions() int { return f.dim }

type fakeStore struct {
	stored  []*Chunk
	deleted []string
}

func (f *fakeStore) Store(_ context.Context, chunks []*Chunk) error {
	f.stored = append(f.stored, chunks...)
	return nil
}

func (f *fakeStore) Search(_ context.Context, _ SearchRequest) ([]ScoredChunk, error) {
	return nil, nil
}

func (f *fakeStore) Delete(_ context.Context, ids []string) error {
	f.deleted = append(f.deleted, ids...)
	return nil
}

type fakeRetriever struct {
	units []RetrievedUnit
	calls int
	lastQ Query
}

func (f *fakeRetriever) Retrieve(_ context.Context, q Query) ([]RetrievedUnit, error) {
	f.calls++
	f.lastQ = q
	return f.units, nil
}

type fakeReranker struct {
	calls int
}

func (f *fakeReranker) Rerank(_ context.Context, query string, units []RetrievedUnit) ([]RetrievedUnit, error) {
	f.calls++
	// Reverse order to prove reranking happened.
	out := make([]RetrievedUnit, len(units))
	for i, u := range units {
		out[len(units)-1-i] = u.WithScore("rerank", float64(len(units)-i))
	}
	return out, nil
}

type fakeAuthorizer struct {
	allowed      map[string]bool
	checkCalls   int
	filterCalls  int
	filterResult ResourceSet
}

func (f *fakeAuthorizer) Check(_ context.Context, req CheckRequest) (bool, error) {
	f.checkCalls++
	return f.allowed[string(req.Resource)], nil
}

func (f *fakeAuthorizer) Filter(_ context.Context, _ FilterRequest) (ResourceSet, error) {
	f.filterCalls++
	return f.filterResult, nil
}

// --- Tests ---

func TestNewPipeline_Validation(t *testing.T) {
	full := PipelineConfig{
		Splitter:  &fakeSplitter{},
		Embedder:  &fakeEmbedder{dim: 3},
		Store:     &fakeStore{},
		Retriever: &fakeRetriever{},
	}

	tests := []struct {
		name    string
		modify  func(*PipelineConfig)
		wantErr bool
	}{
		{"valid config", func(c *PipelineConfig) {}, false},
		{"missing splitter", func(c *PipelineConfig) { c.Splitter = nil }, true},
		{"missing embedder", func(c *PipelineConfig) { c.Embedder = nil }, true},
		{"missing store", func(c *PipelineConfig) { c.Store = nil }, true},
		{"missing retriever", func(c *PipelineConfig) { c.Retriever = nil }, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := full
			tt.modify(&cfg)
			_, err := NewPipeline(cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewPipeline() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPipeline_Ingest(t *testing.T) {
	splitter := &fakeSplitter{
		chunks: []*Chunk{
			NewChunk("c1", "", "hello world", nil),
			NewChunk("c2", "", "foo bar", nil),
		},
	}
	embedder := &fakeEmbedder{dim: 3}
	store := &fakeStore{}

	p, err := NewPipeline(PipelineConfig{
		Splitter:  splitter,
		Embedder:  embedder,
		Store:     store,
		Retriever: &fakeRetriever{},
	})
	if err != nil {
		t.Fatalf("NewPipeline: %v", err)
	}

	doc, err := NewDocument("doc1", TextContent("hello world foo bar", ""))
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}

	ctx := context.Background()
	if err := p.Ingest(ctx, doc); err != nil {
		t.Fatalf("Ingest: %v", err)
	}

	// Verify EmbedBatch was called (not Embed).
	if embedder.batchCalls != 1 {
		t.Errorf("expected 1 EmbedBatch call, got %d", embedder.batchCalls)
	}
	if embedder.embedCalls != 0 {
		t.Errorf("expected 0 Embed calls, got %d", embedder.embedCalls)
	}

	// Verify chunks were stored with vectors.
	if len(store.stored) != 2 {
		t.Fatalf("expected 2 stored chunks, got %d", len(store.stored))
	}
	for i, c := range store.stored {
		if c.Vector == nil {
			t.Errorf("chunk %d has nil vector", i)
		}
		if len(c.Vector) != 3 {
			t.Errorf("chunk %d vector length = %d, want 3", i, len(c.Vector))
		}
		if c.SourceDocumentID() != "doc1" {
			t.Errorf("chunk %d sourceDocID = %q, want %q", i, c.SourceDocumentID(), "doc1")
		}
	}
}

func TestPipeline_Retrieve_Basic(t *testing.T) {
	units := []RetrievedUnit{
		NewChunk("c1", "doc1", "hello", nil),
		NewChunk("c2", "doc2", "world", nil),
	}
	retriever := &fakeRetriever{units: units}

	p, err := NewPipeline(PipelineConfig{
		Splitter:  &fakeSplitter{},
		Embedder:  &fakeEmbedder{dim: 3},
		Store:     &fakeStore{},
		Retriever: retriever,
	})
	if err != nil {
		t.Fatalf("NewPipeline: %v", err)
	}

	ctx := context.Background()
	got, err := p.Retrieve(ctx, Query{Text: "test", TopK: 5})
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}

	if retriever.calls != 1 {
		t.Errorf("expected 1 retriever call, got %d", retriever.calls)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 results, got %d", len(got))
	}
}

func TestPipeline_Retrieve_WithReranker(t *testing.T) {
	units := []RetrievedUnit{
		NewChunk("c1", "doc1", "first", nil),
		NewChunk("c2", "doc2", "second", nil),
	}
	reranker := &fakeReranker{}

	p, err := NewPipeline(PipelineConfig{
		Splitter:  &fakeSplitter{},
		Embedder:  &fakeEmbedder{dim: 3},
		Store:     &fakeStore{},
		Retriever: &fakeRetriever{units: units},
		Reranker:  reranker,
	})
	if err != nil {
		t.Fatalf("NewPipeline: %v", err)
	}

	ctx := context.Background()
	got, err := p.Retrieve(ctx, Query{Text: "test", TopK: 5})
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}

	if reranker.calls != 1 {
		t.Errorf("expected 1 reranker call, got %d", reranker.calls)
	}
	// Reranker reverses order, so c2 should be first.
	if got[0].ID() != "c2" {
		t.Errorf("expected first result to be c2 after reranking, got %s", got[0].ID())
	}
}

func TestPipeline_Retrieve_PostFilterAuth(t *testing.T) {
	units := []RetrievedUnit{
		NewChunk("c1", "doc1", "allowed", nil),
		NewChunk("c2", "doc2", "denied", nil),
		NewChunk("c3", "doc3", "allowed too", nil),
	}
	auth := &fakeAuthorizer{
		allowed: map[string]bool{"doc1": true, "doc3": true},
	}

	p, err := NewPipeline(PipelineConfig{
		Splitter:   &fakeSplitter{},
		Embedder:   &fakeEmbedder{dim: 3},
		Store:      &fakeStore{},
		Retriever:  &fakeRetriever{units: units},
		Authorizer: auth,
		AuthMode:   PostFilter,
	})
	if err != nil {
		t.Fatalf("NewPipeline: %v", err)
	}

	ctx := context.Background()
	got, err := p.Retrieve(ctx, Query{Text: "test", Subject: "user1", TopK: 5})
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}

	if auth.checkCalls != 3 {
		t.Errorf("expected 3 Check calls, got %d", auth.checkCalls)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 authorized results, got %d", len(got))
	}
	if got[0].ID() != "c1" || got[1].ID() != "c3" {
		t.Errorf("unexpected result IDs: %s, %s", got[0].ID(), got[1].ID())
	}
}

func TestPipeline_Retrieve_PreFilterAuth(t *testing.T) {
	units := []RetrievedUnit{
		NewChunk("c1", "doc1", "result", nil),
	}
	auth := &fakeAuthorizer{
		filterResult: ResourceSet{
			Predicate: &MetadataFilter{
				Field: "allowed_docs",
				Op:    FilterOpIn,
				Value: []string{"doc1", "doc3"},
			},
		},
	}
	retriever := &fakeRetriever{units: units}

	p, err := NewPipeline(PipelineConfig{
		Splitter:   &fakeSplitter{},
		Embedder:   &fakeEmbedder{dim: 3},
		Store:      &fakeStore{},
		Retriever:  retriever,
		Authorizer: auth,
		AuthMode:   PreFilter,
	})
	if err != nil {
		t.Fatalf("NewPipeline: %v", err)
	}

	ctx := context.Background()
	got, err := p.Retrieve(ctx, Query{Text: "test", Subject: "user1", TopK: 5})
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}

	if auth.filterCalls != 1 {
		t.Errorf("expected 1 Filter call, got %d", auth.filterCalls)
	}
	if auth.checkCalls != 0 {
		t.Errorf("expected 0 Check calls in PreFilter mode, got %d", auth.checkCalls)
	}
	// Verify the filter was passed through to the query.
	if len(retriever.lastQ.Filters) != 1 {
		t.Errorf("expected 1 filter on query, got %d", len(retriever.lastQ.Filters))
	}
	if len(got) != 1 {
		t.Errorf("expected 1 result, got %d", len(got))
	}
}

func TestPipeline_Delete(t *testing.T) {
	store := &fakeStore{}

	p, err := NewPipeline(PipelineConfig{
		Splitter:  &fakeSplitter{},
		Embedder:  &fakeEmbedder{dim: 3},
		Store:     store,
		Retriever: &fakeRetriever{},
	})
	if err != nil {
		t.Fatalf("NewPipeline: %v", err)
	}

	ctx := context.Background()
	if err := p.Delete(ctx, "doc1", "doc2"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if len(store.deleted) != 2 {
		t.Fatalf("expected 2 deleted IDs, got %d", len(store.deleted))
	}
	if store.deleted[0] != "doc1" || store.deleted[1] != "doc2" {
		t.Errorf("unexpected deleted IDs: %v", store.deleted)
	}
}
