package dory

import "context"

// MetadataFilter is Dory's portable filter expression.
// It is intentionally minimal — just enough to express tenant isolation
// and authorization constraints. Each VectorStore implementation
// translates this into its native query language.
type MetadataFilter struct {
	Field string
	Op    FilterOp
	Value any // string for Eq; []string for In and AnyOf
}

// FilterOp is the comparison operator in a MetadataFilter.
type FilterOp string

const (
	// FilterOpEq matches documents where the field equals the value exactly.
	FilterOpEq FilterOp = "eq"

	// FilterOpIn matches documents where the field equals any value in the list.
	FilterOpIn FilterOp = "in"

	// FilterOpAnyOf matches documents where a metadata array field contains
	// any value from the list. Used for multi-value fields like role lists.
	FilterOpAnyOf FilterOp = "any_of"
)

// SearchRequest bundles everything a VectorStore needs to execute a search.
type SearchRequest struct {
	// QueryVector is the embedding of the user's (possibly transformed) query.
	QueryVector []float32

	// TopK is the maximum number of results to return.
	TopK int

	// Filter, if non-nil, restricts the search to chunks matching
	// these metadata conditions. Tenant isolation and pre-filter
	// authorization constraints are passed here.
	Filter *MetadataFilter
}

// ScoredChunk is a Chunk returned from a vector store search,
// paired with its similarity score.
type ScoredChunk struct {
	Chunk *Chunk
	Score float64
}

// VectorStore is the persistence and similarity search abstraction.
// The library never depends on a concrete implementation — only on this contract.
type VectorStore interface {
	// Store persists a set of chunks. Implementations decide how to
	// physically store the vector, text, and metadata fields.
	Store(ctx context.Context, chunks []*Chunk) error

	// Search finds the top-k chunks whose vectors are nearest to the
	// query vector, applying any metadata filter before scoring.
	Search(ctx context.Context, req SearchRequest) ([]ScoredChunk, error)

	// Delete removes chunks by their IDs. Called on re-ingestion
	// or when a document is permanently removed.
	Delete(ctx context.Context, ids []string) error
}
