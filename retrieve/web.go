package retrieve

import (
	"context"

	"github.com/i33ym/dory"
)

// WebResult represents a single result from a web search.
type WebResult struct {
	URL     string
	Title   string
	Snippet string
}

// WebConfig holds configuration for a Web retriever.
type WebConfig struct {
	// SearchFunc performs a web search and returns results.
	SearchFunc func(ctx context.Context, query string, topK int) ([]WebResult, error)
}

// Web retriever delegates to an external web search function and
// converts the results into Chunk units.
type Web struct {
	config WebConfig
}

// NewWeb creates a new web search retriever.
func NewWeb(config WebConfig) *Web {
	return &Web{config: config}
}

// Retrieve implements [dory.Retriever].
func (w *Web) Retrieve(ctx context.Context, q dory.Query) ([]dory.RetrievedUnit, error) {
	topK := q.TopK
	if topK <= 0 {
		topK = 10
	}

	results, err := w.config.SearchFunc(ctx, q.Text, topK)
	if err != nil {
		return nil, err
	}

	units := make([]dory.RetrievedUnit, len(results))
	for i, r := range results {
		c := dory.NewChunkWithOptions(
			r.URL, // use URL as ID
			"web",
			r.Snippet,
			map[string]any{"title": r.Title},
			r.URL,
			nil,
			0,
		)
		units[i] = c.WithScore("web", 1.0-float64(i)*0.01)
	}
	return units, nil
}
