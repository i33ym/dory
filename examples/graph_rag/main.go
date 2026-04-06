// graph_rag demonstrates graph-based retrieval using GraphFact triples.
// Use this when your queries require reasoning over relationships between
// entities rather than finding semantically similar text fragments.
//
// The graph retriever matches query terms against the Subject, Predicate,
// and Object fields of stored facts. No embeddings or API keys are needed.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/i33ym/dory"
	"github.com/i33ym/dory/retrieve"
)

func main() {
	ctx := context.Background()

	// 1. Create a graph retriever.
	graph := retrieve.NewGraph()

	// 2. Add knowledge graph facts as (subject, predicate, object) triples.
	facts := []*dory.GraphFact{
		dory.NewGraphFact("f1", "doc-001", "Go", "is a", "programming language", nil),
		dory.NewGraphFact("f2", "doc-001", "Dory", "is written in", "Go", nil),
		dory.NewGraphFact("f3", "doc-001", "Dory", "supports", "vector retrieval", nil),
		dory.NewGraphFact("f4", "doc-001", "Dory", "supports", "graph retrieval", nil),
		dory.NewGraphFact("f5", "doc-002", "RAG", "stands for", "Retrieval Augmented Generation", nil),
		dory.NewGraphFact("f6", "doc-002", "RAG", "improves", "LLM accuracy", nil),
		dory.NewGraphFact("f7", "doc-002", "Dory", "implements", "RAG pipelines", nil),
		dory.NewGraphFact("f8", "doc-003", "BM25", "is a", "sparse retrieval algorithm", nil),
		dory.NewGraphFact("f9", "doc-003", "cosine similarity", "measures", "vector distance", nil),
	}

	if err := graph.AddFacts(ctx, facts...); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Added %d facts to the graph.\n\n", len(facts))

	// 3. Query the graph for facts about Dory.
	queries := []string{
		"What does Dory support?",
		"What is Go?",
		"RAG retrieval generation",
	}

	for _, q := range queries {
		fmt.Printf("Query: %q\n", q)
		results, err := graph.Retrieve(ctx, dory.Query{
			Text: q,
			TopK: 3,
		})
		if err != nil {
			log.Fatal(err)
		}

		if len(results) == 0 {
			fmt.Println("  (no matching facts)")
		}
		for i, unit := range results {
			fmt.Printf("  [%d] score=%.4f  %s\n", i+1, unit.Score(), unit.AsText())
		}
		fmt.Println()
	}
}
