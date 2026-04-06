package rerank

import (
	"context"
	"sort"

	dory "github.com/i33ym/dory"
)

// LostInTheMiddle reorders units so the most relevant items appear at
// the edges (beginning and end) of the result list. This is based on
// the finding that LLMs attend more to tokens at the beginning and
// end of their context window, so placing the best results there
// improves downstream answer quality.
//
// It does not change scores — only the ordering.
type LostInTheMiddle struct{}

// NewLostInTheMiddle creates a new LostInTheMiddle reranker.
func NewLostInTheMiddle() *LostInTheMiddle {
	return &LostInTheMiddle{}
}

// Rerank reorders the units so the highest-scored items occupy the
// first and last positions, the next-highest occupy positions 2 and
// N-1, and so on — placing the weakest items in the middle.
func (l *LostInTheMiddle) Rerank(_ context.Context, _ string, units []dory.RetrievedUnit) ([]dory.RetrievedUnit, error) {
	if len(units) == 0 {
		return nil, nil
	}

	// Work on a copy sorted by score descending.
	sorted := make([]dory.RetrievedUnit, len(units))
	copy(sorted, units)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Score() > sorted[j].Score()
	})

	// Alternate placing items at the front and back.
	result := make([]dory.RetrievedUnit, len(sorted))
	front := 0
	back := len(result) - 1

	for i, u := range sorted {
		if i%2 == 0 {
			result[front] = u.WithScore("litm_reorder", u.Score())
			front++
		} else {
			result[back] = u.WithScore("litm_reorder", u.Score())
			back--
		}
	}

	return result, nil
}
