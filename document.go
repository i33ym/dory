package dory

import "time"

// Document is the ingestion unit — a raw source of knowledge before
// it has been chunked or indexed. A document carries its content,
// its identity, and the metadata the authorizer and retriever will
// consult later.
type Document struct {
	// ID uniquely identifies this document within the knowledge base.
	// Every Chunk produced from this document inherits this as its SourceDocumentID.
	ID string

	// Content is the raw text content of the document.
	Content string

	// MimeType describes the format of Content (e.g., "text/markdown",
	// "text/html", "application/pdf"). Chunking strategies may use
	// this to apply format-aware splitting.
	MimeType string

	// Metadata carries arbitrary key-value pairs that will be propagated
	// to every chunk produced from this document. This is where you attach
	// tenant IDs, access control lists, author, creation date, and any
	// other domain-specific attributes.
	Metadata map[string]any

	// CreatedAt records when this document was first ingested.
	CreatedAt time.Time

	// UpdatedAt records the last time this document was re-ingested.
	UpdatedAt time.Time
}
