package eval_test

import (
	"context"
	"fmt"

	"github.com/i33ym/dory"
	"github.com/i33ym/dory/eval"
	"github.com/i33ym/dory/retrieve"
)

func ExampleNewRetrieverEvaluator() {
	// Set up a BM25 retriever with some indexed chunks.
	bm25 := retrieve.NewBM25(retrieve.BM25Config{})
	ctx := context.Background()
	_ = bm25.Index(ctx, []*dory.Chunk{
		dory.NewChunk("c1", "doc-1", "Go is a compiled language", nil),
		dory.NewChunk("c2", "doc-2", "Python is interpreted", nil),
	})

	evaluator, _ := eval.NewRetrieverEvaluator(eval.RetrieverEvaluatorConfig{
		Retriever: bm25,
		TopK:      2,
	})

	results, _ := evaluator.Evaluate(ctx, []dory.TestCase{
		{
			ID:                  "tc-1",
			Question:            "compiled language",
			RelevantDocumentIDs: []string{"doc-1"},
		},
	})

	fmt.Printf("precision: %.1f\n", *results[0].Metrics.ContextPrecision)
	fmt.Printf("recall: %.1f\n", *results[0].Metrics.ContextRecall)
	// Output:
	// precision: 1.0
	// recall: 1.0
}
