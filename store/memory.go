// Package store provides VectorStore implementations for Dory.
// Each implementation satisfies the [dory.VectorStore] interface.
package store

import (
	"context"
	"math"
	"sort"
	"sync"

	"github.com/i33ym/dory"
)

// Memory is an in-memory VectorStore suitable for development and testing.
// It stores chunks in a map and performs brute-force cosine similarity search.
type Memory struct {
	mu     sync.RWMutex
	chunks map[string]*dory.Chunk
}

// NewMemory creates a new in-memory vector store.
func NewMemory() *Memory {
	return &Memory{chunks: make(map[string]*dory.Chunk)}
}

// Store persists chunks in memory.
func (m *Memory) Store(_ context.Context, chunks []*dory.Chunk) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, c := range chunks {
		m.chunks[c.ID()] = c
	}
	return nil
}

// Search finds the top-k chunks by cosine similarity to the query vector.
func (m *Memory) Search(_ context.Context, req dory.SearchRequest) ([]dory.ScoredChunk, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	type scored struct {
		chunk *dory.Chunk
		score float64
	}
	var results []scored

	for _, c := range m.chunks {
		if c.Vector == nil || len(req.QueryVector) == 0 {
			continue
		}
		if req.Filter != nil && !matchFilter(c, req.Filter) {
			continue
		}
		score := cosine(req.QueryVector, c.Vector)
		results = append(results, scored{chunk: c, score: score})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	topK := req.TopK
	if topK > len(results) {
		topK = len(results)
	}

	out := make([]dory.ScoredChunk, topK)
	for i := 0; i < topK; i++ {
		out[i] = dory.ScoredChunk{Chunk: results[i].chunk, Score: results[i].score}
	}
	return out, nil
}

// Delete removes chunks by their IDs.
func (m *Memory) Delete(_ context.Context, ids []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, id := range ids {
		delete(m.chunks, id)
	}
	return nil
}

func matchFilter(c *dory.Chunk, f *dory.MetadataFilter) bool {
	meta := c.Metadata()
	if meta == nil {
		return false
	}
	val, ok := meta[f.Field]
	if !ok {
		return false
	}
	switch f.Op {
	case dory.FilterOpEq:
		return val == f.Value
	default:
		return false
	}
}

func cosine(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
