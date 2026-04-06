// Package retrieve provides retrieval strategy implementations for Dory.
// Each strategy implements the [dory.Retriever] interface.
package retrieve

import (
	"context"

	"github.com/i33ym/dory"
)

// Vector performs dense vector similarity search against a VectorStore.
type Vector struct {
	store    dory.VectorStore
	embedder dory.Embedder
}

// NewVector creates a new vector retriever.
func NewVector(store dory.VectorStore, embedder dory.Embedder) *Vector {
	return &Vector{store: store, embedder: embedder}
}

// Retrieve implements the [dory.Retriever] interface.
func (v *Vector) Retrieve(ctx context.Context, q dory.Query) ([]dory.RetrievedUnit, error) {
	queryVec, err := v.embedder.Embed(ctx, q.Text)
	if err != nil {
		return nil, err
	}

	topK := q.TopK
	if topK <= 0 {
		topK = 10
	}

	req := dory.SearchRequest{
		QueryVector: queryVec,
		TopK:        topK,
	}

	if len(q.Filters) > 0 {
		req.Filter = &q.Filters[0]
	}

	scored, err := v.store.Search(ctx, req)
	if err != nil {
		return nil, err
	}

	units := make([]dory.RetrievedUnit, len(scored))
	for i, sc := range scored {
		units[i] = sc.Chunk.WithScore("vector", sc.Score)
	}
	return units, nil
}
