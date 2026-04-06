// basic_rag demonstrates the simplest possible Dory pipeline:
// fixed-size chunking, OpenAI embeddings, in-memory vector store,
// and vector retrieval. No authorization. No reranking.
// Use this as your starting point before reaching for more complex strategies.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/i33ym/dory"
	"github.com/i33ym/dory/chunk"
	"github.com/i33ym/dory/embed"
	"github.com/i33ym/dory/retrieve"
	"github.com/i33ym/dory/store"
)

func main() {
	ctx := context.Background()

	// 1. Define a document.
	doc, err := dory.NewDocument(
		"doc-001",
		dory.TextContent("Dory is a retrieval intelligence library for Go...", "text/plain"),
		dory.WithMetadata("source", "readme"),
	)
	if err != nil {
		log.Fatal(err)
	}

	// 2. Split the document into chunks.
	splitter := chunk.NewFixed(chunk.FixedConfig{Size: 512, Overlap: 64})
	chunks, err := splitter.Split(ctx, doc)
	if err != nil {
		log.Fatal(err)
	}

	// 3. Embed each chunk and store in the in-memory vector store.
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

	// 4. Retrieve relevant chunks for a query.
	retriever := retrieve.NewVector(vectorStore, embedder)
	results, err := retriever.Retrieve(ctx, dory.Query{
		Text: "What is Dory?",
		TopK: 5,
	})
	if err != nil {
		log.Fatal(err)
	}

	for i, unit := range results {
		fmt.Printf("[%d] score=%.4f\n%s\n\n", i+1, unit.Score(), unit.AsText())
	}
}
