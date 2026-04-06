package dory_test

import (
	"fmt"

	"github.com/i33ym/dory"
)

func ExampleNewDocument() {
	content := dory.TextContent("Hello, Dory!", "text/plain")
	doc, err := dory.NewDocument("doc-1", content,
		dory.WithTenantID("acme"),
		dory.WithLanguage("en"),
		dory.WithMetadata("author", "alice"),
	)
	if err != nil {
		panic(err)
	}
	fmt.Println(doc.ID())
	fmt.Println(doc.TenantID())
	fmt.Println(doc.Language())
	fmt.Println(doc.Metadata()["author"])
	// Output:
	// doc-1
	// acme
	// en
	// alice
}

func ExampleNewChunk() {
	chunk := dory.NewChunk("chunk-1", "doc-1", "The quick brown fox.", nil)
	fmt.Println(chunk.ID())
	fmt.Println(chunk.AsText())
	// Output:
	// chunk-1
	// The quick brown fox.
}

func ExampleTextContent() {
	c := dory.TextContent("some plain text", "")
	text, _ := c.Text()
	fmt.Println(text)
	fmt.Println(c.MimeType())
	fmt.Println(c.Size())
	// Output:
	// some plain text
	// text/plain
	// 15
}
