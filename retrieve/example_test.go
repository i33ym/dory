package retrieve_test

import (
	"context"
	"fmt"

	"github.com/i33ym/dory"
	"github.com/i33ym/dory/retrieve"
)

func ExampleNewBM25() {
	bm25 := retrieve.NewBM25(retrieve.BM25Config{})

	chunks := []*dory.Chunk{
		dory.NewChunk("c1", "doc-1", "Go is a statically typed language", nil),
		dory.NewChunk("c2", "doc-1", "Python is a dynamically typed language", nil),
		dory.NewChunk("c3", "doc-1", "Rust focuses on memory safety", nil),
	}

	ctx := context.Background()
	_ = bm25.Index(ctx, chunks)

	results, _ := bm25.Retrieve(ctx, dory.Query{Text: "typed language", TopK: 2})
	fmt.Println(len(results))
	fmt.Println(results[0].AsText())
	// Output:
	// 2
	// Go is a statically typed language
}
