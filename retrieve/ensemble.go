package retrieve

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/i33ym/dory"
)

// Ensemble retriever runs multiple sub-retrievers concurrently, concatenates
// all results, deduplicates by ID, sorts by score descending, and truncates
// to TopK. Unlike Hybrid, it does not perform any fusion — it simply
// collects and deduplicates.
type Ensemble struct {
	retrievers []dory.Retriever
}

// NewEnsemble creates a new ensemble retriever.
func NewEnsemble(retrievers []dory.Retriever) *Ensemble {
	return &Ensemble{retrievers: retrievers}
}

// Retrieve implements [dory.Retriever].
func (e *Ensemble) Retrieve(ctx context.Context, q dory.Query) ([]dory.RetrievedUnit, error) {
	ch := make(chan retrieverResult, len(e.retrievers))
	var wg sync.WaitGroup

	for i, r := range e.retrievers {
		wg.Add(1)
		go func(idx int, ret dory.Retriever) {
			defer wg.Done()
			results, err := ret.Retrieve(ctx, q)
			ch <- retrieverResult{index: idx, results: results, err: err}
		}(i, r)
	}

	wg.Wait()
	close(ch)

	// Collect results and check for errors.
	allResults := make([][]dory.RetrievedUnit, len(e.retrievers))
	for res := range ch {
		if res.err != nil {
			return nil, fmt.Errorf("retriever %d: %w", res.index, res.err)
		}
		allResults[res.index] = res.results
	}

	// Deduplicate by ID, keeping the first occurrence (highest score from
	// whichever retriever returned it first).
	seen := make(map[string]bool)
	var deduped []dory.RetrievedUnit

	for _, results := range allResults {
		for _, unit := range results {
			id := unit.ID()
			if seen[id] {
				continue
			}
			seen[id] = true
			deduped = append(deduped, unit.WithScore("ensemble", unit.Score()))
		}
	}

	// Sort by score descending.
	sort.Slice(deduped, func(i, j int) bool {
		return deduped[i].Score() > deduped[j].Score()
	})

	topK := q.TopK
	if topK <= 0 {
		topK = 10
	}
	if len(deduped) > topK {
		deduped = deduped[:topK]
	}

	return deduped, nil
}
