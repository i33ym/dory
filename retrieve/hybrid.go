package retrieve

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/i33ym/dory"
)

// HybridConfig holds parameters for hybrid retrieval with RRF.
type HybridConfig struct {
	// K is the RRF constant. Default 60.
	K int
}

func (c HybridConfig) withDefaults() HybridConfig {
	if c.K == 0 {
		c.K = 60
	}
	return c
}

// Hybrid retriever calls multiple sub-retrievers concurrently and fuses
// their results using Reciprocal Rank Fusion (RRF).
type Hybrid struct {
	retrievers []dory.Retriever
	config     HybridConfig
}

// NewHybrid creates a new hybrid retriever.
func NewHybrid(retrievers []dory.Retriever, config HybridConfig) *Hybrid {
	config = config.withDefaults()
	return &Hybrid{
		retrievers: retrievers,
		config:     config,
	}
}

// retrieverResult holds the output from one sub-retriever.
type retrieverResult struct {
	index   int
	results []dory.RetrievedUnit
	err     error
}

// Retrieve implements [dory.Retriever] using RRF fusion.
func (h *Hybrid) Retrieve(ctx context.Context, q dory.Query) ([]dory.RetrievedUnit, error) {
	// Run all retrievers concurrently.
	ch := make(chan retrieverResult, len(h.retrievers))
	var wg sync.WaitGroup

	for i, r := range h.retrievers {
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
	allResults := make([][]dory.RetrievedUnit, len(h.retrievers))
	for res := range ch {
		if res.err != nil {
			return nil, fmt.Errorf("retriever %d: %w", res.index, res.err)
		}
		allResults[res.index] = res.results
	}

	// Compute RRF scores. Deduplicate by unit ID, keeping the first
	// concrete unit encountered for each ID.
	type fusedEntry struct {
		unit  dory.RetrievedUnit
		score float64
	}
	fused := make(map[string]*fusedEntry)
	k := float64(h.config.K)

	for _, results := range allResults {
		for rank, unit := range results {
			id := unit.ID()
			rrfContribution := 1.0 / (k + float64(rank+1))
			if entry, ok := fused[id]; ok {
				entry.score += rrfContribution
			} else {
				fused[id] = &fusedEntry{
					unit:  unit,
					score: rrfContribution,
				}
			}
		}
	}

	// Sort by fused score descending.
	sorted := make([]*fusedEntry, 0, len(fused))
	for _, entry := range fused {
		sorted = append(sorted, entry)
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].score > sorted[j].score
	})

	topK := q.TopK
	if topK <= 0 {
		topK = 10
	}
	if len(sorted) > topK {
		sorted = sorted[:topK]
	}

	units := make([]dory.RetrievedUnit, len(sorted))
	for i, entry := range sorted {
		units[i] = entry.unit.WithScore("rrf_fusion", entry.score)
	}
	return units, nil
}
