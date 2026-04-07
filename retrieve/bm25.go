package retrieve

import (
	"context"
	"math"
	"sort"
	"strings"
	"sync"

	"github.com/i33ym/dory"
	"github.com/i33ym/dory/internal/filter"
)

// BM25Config holds parameters for BM25 scoring.
type BM25Config struct {
	// K1 controls term-frequency saturation. Default 1.2.
	K1 float64

	// B controls length normalization. Default 0.75.
	B float64
}

func (c BM25Config) withDefaults() BM25Config {
	if c.K1 == 0 {
		c.K1 = 1.2
	}
	if c.B == 0 {
		c.B = 0.75
	}
	return c
}

// indexedDoc stores the pre-computed token frequencies for a single chunk.
type indexedDoc struct {
	chunk  *dory.Chunk
	tf     map[string]int // term -> raw count
	length int            // total token count
}

// BM25 is an in-memory sparse retriever using the Okapi BM25 scoring function.
type BM25 struct {
	mu     sync.RWMutex
	config BM25Config

	docs   []indexedDoc
	df     map[string]int // term -> number of docs containing it
	avgLen float64        // average document length in tokens
}

// NewBM25 creates a new BM25 sparse retriever.
func NewBM25(config BM25Config) *BM25 {
	config = config.withDefaults()
	return &BM25{
		config: config,
		df:     make(map[string]int),
	}
}

// tokenize splits text into lowercase whitespace-delimited tokens.
func tokenize(text string) []string {
	return strings.Fields(strings.ToLower(text))
}

// Index adds chunks to the BM25 index. It may be called multiple times.
func (b *BM25) Index(_ context.Context, chunks []*dory.Chunk) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, c := range chunks {
		tokens := tokenize(c.AsText())
		tf := make(map[string]int, len(tokens))
		for _, t := range tokens {
			tf[t]++
		}

		// Update document frequency for each unique term.
		for term := range tf {
			b.df[term]++
		}

		b.docs = append(b.docs, indexedDoc{
			chunk:  c,
			tf:     tf,
			length: len(tokens),
		})
	}

	// Recompute average document length.
	total := 0
	for _, d := range b.docs {
		total += d.length
	}
	if len(b.docs) > 0 {
		b.avgLen = float64(total) / float64(len(b.docs))
	}

	return nil
}

// Retrieve implements [dory.Retriever] using BM25 scoring.
func (b *BM25) Retrieve(_ context.Context, q dory.Query) ([]dory.RetrievedUnit, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if len(b.docs) == 0 {
		return nil, nil
	}

	topK := q.TopK
	if topK <= 0 {
		topK = 10
	}

	queryTerms := tokenize(q.Text)
	n := float64(len(b.docs))

	type scored struct {
		idx   int
		score float64
	}

	results := make([]scored, 0, len(b.docs))
	for i, doc := range b.docs {
		if !filter.MatchAll(doc.chunk.Metadata(), q.Filters) {
			continue
		}
		var score float64
		for _, term := range queryTerms {
			tfRaw, ok := doc.tf[term]
			if !ok {
				continue
			}

			dfVal := float64(b.df[term])
			// IDF with the standard BM25 formula (Robertson variant).
			idf := math.Log(1 + (n-dfVal+0.5)/(dfVal+0.5))

			tf := float64(tfRaw)
			denom := tf + b.config.K1*(1-b.config.B+b.config.B*float64(doc.length)/b.avgLen)
			score += idf * (tf * (b.config.K1 + 1)) / denom
		}
		if score > 0 {
			results = append(results, scored{idx: i, score: score})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	if len(results) > topK {
		results = results[:topK]
	}

	units := make([]dory.RetrievedUnit, len(results))
	for i, r := range results {
		units[i] = b.docs[r.idx].chunk.WithScore("bm25", r.score)
	}
	return units, nil
}
