package retrieve

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/i33ym/dory"
)

// Graph is an in-memory graph store for GraphFact triples. It provides
// simple text matching retrieval: query terms are matched against the
// Subject, Predicate, and Object fields of each stored fact.
type Graph struct {
	mu    sync.RWMutex
	facts []*dory.GraphFact
}

// NewGraph creates a new in-memory graph retriever.
func NewGraph() *Graph {
	return &Graph{}
}

// AddFacts stores one or more GraphFact triples.
func (g *Graph) AddFacts(_ context.Context, facts ...*dory.GraphFact) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.facts = append(g.facts, facts...)
	return nil
}

// Retrieve implements [dory.Retriever]. It searches for query terms
// in the Subject, Predicate, and Object fields of each stored fact.
// The score is the fraction of query terms that matched.
func (g *Graph) Retrieve(_ context.Context, q dory.Query) ([]dory.RetrievedUnit, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	terms := strings.Fields(strings.ToLower(q.Text))
	if len(terms) == 0 {
		return nil, nil
	}

	type scored struct {
		fact  *dory.GraphFact
		score float64
	}

	var matches []scored
	for _, fact := range g.facts {
		combined := strings.ToLower(fact.Subject + " " + fact.Predicate + " " + fact.Object)
		matchCount := 0
		for _, term := range terms {
			if strings.Contains(combined, term) {
				matchCount++
			}
		}
		if matchCount > 0 {
			score := float64(matchCount) / float64(len(terms))
			matches = append(matches, scored{fact: fact, score: score})
		}
	}

	// Sort by score descending.
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].score > matches[j].score
	})

	topK := q.TopK
	if topK <= 0 {
		topK = 10
	}
	if len(matches) > topK {
		matches = matches[:topK]
	}

	units := make([]dory.RetrievedUnit, len(matches))
	for i, m := range matches {
		units[i] = m.fact.WithScore("graph", m.score)
	}
	return units, nil
}
