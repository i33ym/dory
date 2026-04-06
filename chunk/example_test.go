package chunk_test

import (
	"context"
	"fmt"

	"github.com/i33ym/dory"
	"github.com/i33ym/dory/chunk"
)

func ExampleNewFixed() {
	doc, _ := dory.NewDocument("doc-1", dory.TextContent("Hello world, this is a test of fixed chunking.", ""))
	splitter := chunk.NewFixed(chunk.FixedConfig{Size: 20})
	chunks, _ := splitter.Split(context.Background(), doc)
	fmt.Println(len(chunks))
	fmt.Println(chunks[0].Text())
	// Output:
	// 3
	// Hello world, this is
}

func ExampleNewRecursive() {
	text := "First paragraph.\n\nSecond paragraph.\n\nThird paragraph."
	doc, _ := dory.NewDocument("doc-1", dory.TextContent(text, ""))
	splitter := chunk.NewRecursive(chunk.RecursiveConfig{Size: 30})
	chunks, _ := splitter.Split(context.Background(), doc)
	fmt.Println(len(chunks))
	fmt.Println(chunks[0].Text())
	// Output:
	// 3
	// First paragraph.
}
