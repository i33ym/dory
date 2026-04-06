// Package rerank provides reranking implementations for Dory.
// Each implementation satisfies the [dory.Reranker] interface.
package rerank

import (
	"context"
	"sort"
	"sync"

	"golang.org/x/sync/errgroup"

	dory "github.com/i33ym/dory"
)

// CrossEncoderConfig configures a CrossEncoder reranker.
type CrossEncoderConfig struct {
	// ScoreFunc scores a single query-document pair and returns a
	// relevance score. This is typically backed by a cross-encoder
	// model API call.
	ScoreFunc func(ctx context.Context, query string, document string) (float64, error)

	// TopK limits the number of results returned. Zero means return all.
	TopK int

	// Threshold is the minimum score a unit must reach to be included
	// in the results. Defaults to 0.
	Threshold float64
}

// CrossEncoder reranks retrieved units by scoring each query-document
// pair with a cross-encoder model, then sorting by the new scores.
type CrossEncoder struct {
	scoreFunc func(ctx context.Context, query string, document string) (float64, error)
	topK      int
	threshold float64
}

// NewCrossEncoder creates a new CrossEncoder reranker from the given config.
func NewCrossEncoder(config CrossEncoderConfig) *CrossEncoder {
	return &CrossEncoder{
		scoreFunc: config.ScoreFunc,
		topK:      config.TopK,
		threshold: config.Threshold,
	}
}

// Rerank scores each unit concurrently against the query using the
// configured ScoreFunc, sorts by descending score, and applies TopK
// and Threshold filtering.
func (ce *CrossEncoder) Rerank(ctx context.Context, query string, units []dory.RetrievedUnit) ([]dory.RetrievedUnit, error) {
	if len(units) == 0 {
		return nil, nil
	}

	type scored struct {
		unit  dory.RetrievedUnit
		score float64
	}

	results := make([]scored, len(units))
	var mu sync.Mutex

	g, gctx := errgroup.WithContext(ctx)
	for i, u := range units {
		i, u := i, u
		g.Go(func() error {
			s, err := ce.scoreFunc(gctx, query, u.AsText())
			if err != nil {
				return err
			}
			mu.Lock()
			results[i] = scored{unit: u, score: s}
			mu.Unlock()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	// Sort by score descending.
	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	// Apply threshold and TopK.
	out := make([]dory.RetrievedUnit, 0, len(results))
	for _, r := range results {
		if r.score < ce.threshold {
			continue
		}
		out = append(out, r.unit.WithScore("crossencoder", r.score))
		if ce.topK > 0 && len(out) >= ce.topK {
			break
		}
	}

	return out, nil
}
