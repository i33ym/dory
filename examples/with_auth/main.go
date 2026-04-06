// with_auth demonstrates Dory's authorization integration using
// the Allowlist backend in PostFilter mode. Documents that the
// caller is not authorized to see are filtered out after retrieval.
//
// No API keys or external services are needed — this example uses
// BM25 sparse retrieval with an in-memory allowlist authorizer.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/i33ym/dory"
	"github.com/i33ym/dory/auth"
	"github.com/i33ym/dory/retrieve"
)

func main() {
	ctx := context.Background()

	// 1. Create chunks from different documents with different access levels.
	chunks := []*dory.Chunk{
		dory.NewChunk("c1", "doc-public", "Dory is an open-source retrieval library for Go.", nil),
		dory.NewChunk("c2", "doc-public", "Dory supports vector, BM25, hybrid, and graph retrieval.", nil),
		dory.NewChunk("c3", "doc-internal", "Internal roadmap: add support for pgvector and Qdrant backends.", nil),
		dory.NewChunk("c4", "doc-internal", "Internal design: the pipeline uses composable interfaces.", nil),
		dory.NewChunk("c5", "doc-secret", "Secret project: real-time streaming retrieval with WebSocket.", nil),
	}

	// 2. Index all chunks in BM25 (no embeddings needed).
	bm25 := retrieve.NewBM25(retrieve.BM25Config{})
	if err := bm25.Index(ctx, chunks); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Indexed %d chunks across 3 documents.\n\n", len(chunks))

	// 3. Set up an allowlist authorizer with grants per user.
	authorizer := auth.NewAllowlist(auth.AllowlistConfig{
		Grants: map[string][]string{
			"alice": {"doc-public", "doc-internal"},               // can see public + internal
			"bob":   {"doc-public"},                               // can see public only
			"carol": {"doc-public", "doc-internal", "doc-secret"}, // can see everything
		},
	})

	// 4. Retrieve for each user using PostFilter authorization.
	//    The retriever returns all matches, then the authorizer filters out
	//    documents the user is not allowed to see.
	query := dory.Query{
		Text: "Dory retrieval support",
		TopK: 10,
	}

	users := []string{"alice", "bob", "carol"}
	for _, user := range users {
		fmt.Printf("=== Results for %s ===\n", user)

		// Retrieve all matching chunks.
		results, err := bm25.Retrieve(ctx, query)
		if err != nil {
			log.Fatal(err)
		}

		// Post-filter: check each result against the authorizer.
		var allowed []dory.RetrievedUnit
		for _, unit := range results {
			ok, err := authorizer.Check(ctx, dory.CheckRequest{
				Subject:  dory.Subject(user),
				Action:   dory.ActionRead,
				Resource: dory.Resource(unit.SourceDocumentID()),
			})
			if err != nil {
				log.Fatal(err)
			}
			if ok {
				allowed = append(allowed, unit)
			}
		}

		if len(allowed) == 0 {
			fmt.Println("  (no authorized results)")
		}
		for i, unit := range allowed {
			fmt.Printf("  [%d] doc=%s score=%.4f\n      %s\n",
				i+1, unit.SourceDocumentID(), unit.Score(), unit.AsText())
		}
		fmt.Println()
	}
}
