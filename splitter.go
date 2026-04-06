package dory

import "context"

// Splitter transforms a Document into a sequence of Chunks.
// Each concrete implementation in the chunk/ sub-package represents
// a different strategy for finding good chunk boundaries.
type Splitter interface {
	// Split takes a Document and returns the chunks produced from it.
	// Implementations must propagate doc.ID as each chunk's SourceDocumentID
	// and doc.Metadata as the base for each chunk's metadata.
	Split(ctx context.Context, doc *Document) ([]*Chunk, error)
}
