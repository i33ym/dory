package dory

import (
	"context"
	"log"
)

// Hook is called at specific points in the pipeline lifecycle.
// Hooks observe but do not modify pipeline behavior.
type Hook struct {
	// BeforeIngest is called before documents are ingested.
	// Receives the number of documents about to be processed.
	BeforeIngest func(ctx context.Context, docCount int)

	// AfterIngest is called after documents are ingested.
	// Receives the number of chunks produced and any error.
	AfterIngest func(ctx context.Context, chunkCount int, err error)

	// BeforeRetrieve is called before a retrieval query is executed.
	BeforeRetrieve func(ctx context.Context, query Query)

	// AfterRetrieve is called after retrieval completes.
	// Receives the number of results and any error.
	AfterRetrieve func(ctx context.Context, resultCount int, err error)

	// BeforeRerank is called before reranking.
	BeforeRerank func(ctx context.Context, query string, candidateCount int)

	// AfterRerank is called after reranking completes.
	AfterRerank func(ctx context.Context, resultCount int, err error)
}

// NewLogHook creates a Hook that logs pipeline events using the standard log package.
func NewLogHook() Hook {
	return Hook{
		BeforeIngest: func(_ context.Context, docCount int) {
			log.Printf("dory: ingesting %d documents", docCount)
		},
		AfterIngest: func(_ context.Context, chunkCount int, err error) {
			if err != nil {
				log.Printf("dory: ingest failed: %v", err)
			} else {
				log.Printf("dory: ingested %d chunks", chunkCount)
			}
		},
		BeforeRetrieve: func(_ context.Context, query Query) {
			log.Printf("dory: retrieving query=%q topK=%d", query.Text, query.TopK)
		},
		AfterRetrieve: func(_ context.Context, resultCount int, err error) {
			if err != nil {
				log.Printf("dory: retrieve failed: %v", err)
			} else {
				log.Printf("dory: retrieved %d results", resultCount)
			}
		},
		BeforeRerank: func(_ context.Context, query string, candidateCount int) {
			log.Printf("dory: reranking %d candidates", candidateCount)
		},
		AfterRerank: func(_ context.Context, resultCount int, err error) {
			if err != nil {
				log.Printf("dory: rerank failed: %v", err)
			} else {
				log.Printf("dory: reranked to %d results", resultCount)
			}
		},
	}
}
