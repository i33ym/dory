package dory

import "context"

// Reranker reorders a slice of RetrievedUnits by their relevance
// to the original query. It operates after initial retrieval, trading
// latency for precision.
type Reranker interface {
	// Rerank takes the original query text and the candidate units
	// returned by the retriever, and returns them in a new order
	// with updated scores. The returned slice may be shorter than
	// the input if the reranker applies a relevance threshold.
	Rerank(ctx context.Context, query string, units []RetrievedUnit) ([]RetrievedUnit, error)
}
