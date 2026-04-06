// hybrid_rag demonstrates hybrid retrieval combining dense vector search
// with BM25 sparse retrieval, fused via Reciprocal Rank Fusion (RRF).
// This consistently outperforms either approach alone, especially for
// queries that mix natural language intent with specific identifiers or keywords.
//
// NOTE: This example requires OPENAI_API_KEY to be set for real embedding.
// Without it, the program demonstrates the wiring with a placeholder message.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/i33ym/dory"
	"github.com/i33ym/dory/chunk"
	"github.com/i33ym/dory/embed"
	"github.com/i33ym/dory/retrieve"
	"github.com/i33ym/dory/store"
)

func main() {
	ctx := context.Background()

	// 1. Define a document with enough text to produce multiple chunks.
	doc, err := dory.NewDocument(
		"doc-001",
		dory.TextContent(
			"Dory is a retrieval intelligence library for Go. "+
				"It supports vector search, BM25 sparse retrieval, and hybrid fusion. "+
				"Hybrid retrieval combines the semantic understanding of dense embeddings "+
				"with the keyword precision of BM25 scoring. "+
				"Reciprocal Rank Fusion merges ranked lists from multiple retrievers "+
				"into a single result set that outperforms either source alone. "+
				"Dory's pipeline is composable: chunking, embedding, indexing, "+
				"retrieval, reranking, and authorization are all pluggable interfaces.",
			"text/plain",
		),
		dory.WithMetadata("source", "docs"),
	)
	if err != nil {
		log.Fatal(err)
	}

	// 2. Split the document into chunks.
	splitter := chunk.NewFixed(chunk.FixedConfig{Size: 128, Overlap: 32})
	chunks, err := splitter.Split(ctx, doc)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Split document into %d chunks.\n", len(chunks))

	// 3. Index chunks in BM25 (no embeddings needed for sparse search).
	bm25 := retrieve.NewBM25(retrieve.BM25Config{})
	if err := bm25.Index(ctx, chunks); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Indexed chunks in BM25.")

	// 4. For the vector path, we need an embedder and a vector store.
	//    Check if OPENAI_API_KEY is available.
	if os.Getenv("OPENAI_API_KEY") == "" {
		fmt.Println()
		fmt.Println("OPENAI_API_KEY is not set. Demonstrating BM25-only retrieval.")
		fmt.Println("Set OPENAI_API_KEY to see full hybrid (vector + BM25) retrieval.")
		fmt.Println()

		// Show BM25-only results as a demonstration.
		results, err := bm25.Retrieve(ctx, dory.Query{
			Text: "hybrid fusion retrieval",
			TopK: 3,
		})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("BM25 results:")
		for i, unit := range results {
			fmt.Printf("  [%d] score=%.4f\n      %s\n\n", i+1, unit.Score(), unit.AsText())
		}
		return
	}

	// 5. Embed each chunk and store in memory.
	embedder := embed.NewOpenAI("text-embedding-3-small")
	vectorStore := store.NewMemory()

	for _, c := range chunks {
		vec, err := embedder.Embed(ctx, c.AsText())
		if err != nil {
			log.Fatal(err)
		}
		c.Vector = vec
	}
	if err := vectorStore.Store(ctx, chunks); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Embedded and stored chunks in vector store.")

	// 6. Create a hybrid retriever combining vector and BM25.
	vectorRetriever := retrieve.NewVector(vectorStore, embedder)
	hybrid := retrieve.NewHybrid(
		[]dory.Retriever{vectorRetriever, bm25},
		retrieve.HybridConfig{K: 60},
	)

	// 7. Retrieve with the hybrid retriever.
	results, err := hybrid.Retrieve(ctx, dory.Query{
		Text: "hybrid fusion retrieval",
		TopK: 3,
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Hybrid (vector + BM25) results:")
	for i, unit := range results {
		fmt.Printf("  [%d] score=%.4f\n      %s\n\n", i+1, unit.Score(), unit.AsText())
	}
}
