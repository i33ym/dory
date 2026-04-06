package dory

import (
	"fmt"
	"strings"
)

// RetrievedUnit is the common interface for everything Dory can retrieve,
// regardless of which retrieval strategy produced it. The pipeline —
// reranking, authorization, and prompt injection — works exclusively
// against this interface, remaining agnostic about the concrete type.
type RetrievedUnit interface {
	// ID returns a stable unique identifier for this unit.
	// Used for deduplication when multiple retrievers return the same content.
	ID() string

	// SourceDocumentID returns the document or resource this unit came from.
	// The authorizer uses this to check permissions without needing to know
	// whether the unit is a chunk, a graph fact, or a database row.
	SourceDocumentID() string

	// AsText returns a natural language representation of this unit
	// suitable for injection into an LLM prompt. Every unit must be
	// expressible as text, even if its native representation is structured.
	AsText() string

	// Score returns the relevance score assigned by the retriever.
	// The reranker updates this score after cross-encoder evaluation.
	Score() float64

	// WithScore returns a copy of this unit with the given score.
	// Used by rerankers to update relevance without mutating the original.
	WithScore(score float64) RetrievedUnit

	// Metadata returns arbitrary key-value pairs attached to this unit.
	// The authorizer's pre-filter metadata lives here.
	Metadata() map[string]any
}

// Chunk is the concrete RetrievedUnit for text-based retrieval strategies:
// vector search, sparse search, hybrid search, and their variants.
// This is the type that chunking strategies produce and that most
// of Dory's pipeline is optimized for.
type Chunk struct {
	id          string
	sourceDocID string
	text        string
	score       float64
	metadata    map[string]any

	// Vector is the dense embedding of this chunk's text.
	// Nil until the embedder processes this chunk.
	Vector []float32

	// ParentID, if non-empty, points to the larger parent chunk
	// this chunk was derived from. This is what makes small-to-big
	// retrieval possible: retrieve the child for its precise embedding,
	// then return the parent for richer context.
	ParentID string

	// WindowText, if non-empty, is the surrounding sentence window
	// (sentences before and after this chunk's core text).
	// When set, AsText returns this instead of the raw chunk text,
	// giving the LLM more coherent context.
	WindowText string

	// ContextPrefix is a short LLM-generated sentence that situates
	// this chunk within its source document. Prepended by the
	// contextual retrieval technique before embedding.
	ContextPrefix string
}

func (c *Chunk) ID() string               { return c.id }
func (c *Chunk) SourceDocumentID() string { return c.sourceDocID }
func (c *Chunk) Score() float64           { return c.score }
func (c *Chunk) Metadata() map[string]any { return c.metadata }

func (c *Chunk) AsText() string {
	if c.WindowText != "" {
		return c.WindowText
	}
	if c.ContextPrefix != "" {
		return c.ContextPrefix + " " + c.text
	}
	return c.text
}

func (c *Chunk) WithScore(score float64) RetrievedUnit {
	// Return a shallow copy with the new score.
	// We copy rather than mutate so rerankers can work safely
	// on slices from concurrent retrievers.
	cp := *c
	cp.score = score
	return &cp
}

// NewChunk constructs a Chunk with the required identity fields.
func NewChunk(id, sourceDocID, text string, metadata map[string]any) *Chunk {
	return &Chunk{
		id:          id,
		sourceDocID: sourceDocID,
		text:        text,
		metadata:    metadata,
	}
}

// GraphFact is the concrete RetrievedUnit for graph retrieval.
// It represents a single fact extracted from the knowledge graph:
// a subject, a predicate (relationship type), and an object.
// Example: Subject="Elon Musk", Predicate="CEO_OF", Object="Tesla".
type GraphFact struct {
	id          string
	sourceDocID string
	score       float64
	metadata    map[string]any

	Subject   string
	Predicate string
	Object    string
}

func (g *GraphFact) ID() string               { return g.id }
func (g *GraphFact) SourceDocumentID() string { return g.sourceDocID }
func (g *GraphFact) Score() float64           { return g.score }
func (g *GraphFact) Metadata() map[string]any { return g.metadata }

// AsText verbalizes the graph triple into natural language suitable
// for LLM prompt injection.
func (g *GraphFact) AsText() string {
	return fmt.Sprintf("%s is related to %s via %s.", g.Subject, g.Object, g.Predicate)
}

func (g *GraphFact) WithScore(score float64) RetrievedUnit {
	cp := *g
	cp.score = score
	return &cp
}

// StructuredRow is the concrete RetrievedUnit for structured retrieval —
// the case where the knowledge base is a database and the retriever
// executed a generated SQL query. Each row from the result set becomes
// one StructuredRow.
type StructuredRow struct {
	id          string
	sourceDocID string
	score       float64
	metadata    map[string]any

	// Columns preserves the relational structure of the row,
	// keyed by column name.
	Columns map[string]any
}

func (s *StructuredRow) ID() string               { return s.id }
func (s *StructuredRow) SourceDocumentID() string { return s.sourceDocID }
func (s *StructuredRow) Score() float64           { return s.score }
func (s *StructuredRow) Metadata() map[string]any { return s.metadata }

// AsText serializes the row into a readable key-value sentence.
func (s *StructuredRow) AsText() string {
	parts := make([]string, 0, len(s.Columns))
	for k, v := range s.Columns {
		parts = append(parts, fmt.Sprintf("%s: %v", k, v))
	}
	var result strings.Builder
	for i, p := range parts {
		if i > 0 {
			result.WriteString(", ")
		}
		result.WriteString(p)
	}
	return "[row] " + result.String()
}

func (s *StructuredRow) WithScore(score float64) RetrievedUnit {
	cp := *s
	cp.score = score
	return &cp
}
