package retrieve

import (
	"context"

	"github.com/i33ym/dory"
)

// Route binds a named retriever to a matching function.
type Route struct {
	// Name is a human-readable label for this route.
	Name string

	// Retriever is the retriever to use when this route matches.
	Retriever dory.Retriever

	// Match returns true if the given query should be handled by this route.
	Match func(q dory.Query) bool
}

// RouterConfig holds configuration for a Router retriever.
type RouterConfig struct {
	Routes []Route
}

// Router retriever routes queries to the best retriever based on a
// routing function. It iterates routes in order and uses the first
// matching retriever.
type Router struct {
	routes []Route
}

// NewRouter creates a new router retriever.
func NewRouter(config RouterConfig) *Router {
	return &Router{routes: config.Routes}
}

// Retrieve implements [dory.Retriever].
func (r *Router) Retrieve(ctx context.Context, q dory.Query) ([]dory.RetrievedUnit, error) {
	for _, route := range r.routes {
		if route.Match(q) {
			return route.Retriever.Retrieve(ctx, q)
		}
	}
	return nil, nil
}
