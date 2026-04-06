package retrieve

import (
	"context"

	"github.com/i33ym/dory"
)

// SmallToBigConfig configures the small-to-big retrieval strategy.
type SmallToBigConfig struct {
	// Retriever is the child retriever that returns small, precise chunks.
	Retriever dory.Retriever

	// Parents maps parent chunk IDs to their full parent chunks.
	// Populated during ingestion so the retriever can expand
	// child chunks to their larger parents.
	Parents map[string]*dory.Chunk
}

// SmallToBig retrieves small chunks for precise embedding matches,
// then expands them to their parent chunks for richer LLM context.
type SmallToBig struct {
	config SmallToBigConfig
}

// NewSmallToBig creates a new small-to-big retriever.
func NewSmallToBig(config SmallToBigConfig) *SmallToBig {
	return &SmallToBig{config: config}
}

// Retrieve implements the [dory.Retriever] interface.
// It calls the child retriever, then replaces each child chunk with
// its parent chunk (if one exists). Results are deduplicated by parent ID,
// keeping the highest child score for each parent.
func (s *SmallToBig) Retrieve(ctx context.Context, q dory.Query) ([]dory.RetrievedUnit, error) {
	children, err := s.config.Retriever.Retrieve(ctx, q)
	if err != nil {
		return nil, err
	}

	// Track best score per parent to deduplicate.
	seen := make(map[string]int) // parentID -> index in results
	var results []dory.RetrievedUnit

	for _, child := range children {
		chunk, ok := child.(*dory.Chunk)
		if !ok || chunk.ParentID == "" {
			// Not a chunk or no parent — pass through as-is.
			results = append(results, child)
			continue
		}

		parent, found := s.config.Parents[chunk.ParentID]
		if !found {
			// Parent not in map — pass through the child.
			results = append(results, child)
			continue
		}

		if idx, dup := seen[chunk.ParentID]; dup {
			// Already have this parent; keep the higher score.
			if child.Score() > results[idx].Score() {
				results[idx] = parent.WithScore("small_to_big", child.Score())
			}
			continue
		}

		seen[chunk.ParentID] = len(results)
		results = append(results, parent.WithScore("small_to_big", child.Score()))
	}

	return results, nil
}
