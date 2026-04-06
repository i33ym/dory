package dory

import "context"

// Query carries everything a retriever needs to find relevant units.
type Query struct {
	// Text is the raw natural language question from the user.
	Text string

	// TenantID is mandatory for multi-tenant knowledge bases.
	// Retrievers must enforce tenant isolation before any other
	// filtering. An empty TenantID is valid only for single-tenant systems.
	TenantID string

	// Subject is the identity of the caller for authorization checks.
	// Passed to the Authorizer when authorization is enabled.
	Subject string

	// TopK is the maximum number of results the caller wants.
	// Retrievers may internally over-fetch (e.g., for reranking)
	// but should return at most TopK results.
	TopK int

	// Filters are additional metadata constraints the caller wants
	// applied beyond tenant isolation and authorization.
	Filters []MetadataFilter
}

// Retriever finds the most relevant RetrievedUnits for a Query.
// All retrieval strategies — vector, sparse, hybrid, graph, structured,
// web — implement this interface.
type Retriever interface {
	Retrieve(ctx context.Context, q Query) ([]RetrievedUnit, error)
}
