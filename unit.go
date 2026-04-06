package dory

import (
	"encoding/json"
	"fmt"
	"maps"
	"strings"
)

// ScoreEntry records a single scoring event in a unit's retrieval history.
type ScoreEntry struct {
	// Stage is the name of the pipeline stage that assigned this score.
	// Examples: "vector", "bm25", "rrf_fusion", "crossencoder", "final".
	Stage string `json:"stage"`

	// Score is the relevance score assigned at this stage.
	Score float64 `json:"score"`
}

// Position describes where in the source document a chunk came from.
type Position struct {
	// StartByte and EndByte are the byte offsets in the original
	// document content. Used for precise deduplication and
	// for reconstructing the original context.
	StartByte int `json:"start_byte"`
	EndByte   int `json:"end_byte"`

	// Page is the page number in a paginated document (PDF, DOCX).
	// Nil for documents without pagination.
	Page *int `json:"page,omitempty"`

	// Section is the heading path to this chunk's location in a
	// structured document. For a Markdown file:
	// ["Introduction", "Background", "Prior Work"].
	// Nil for unstructured documents.
	Section []string `json:"section,omitempty"`
}

// RetrievedUnit is the common interface for everything Dory can retrieve,
// regardless of which retrieval strategy produced it. The pipeline —
// reranking, authorization, and prompt injection — works exclusively
// against this interface, remaining agnostic about the concrete type.
type RetrievedUnit interface {
	// ID returns a stable unique identifier for this unit.
	ID() string

	// SourceDocumentID returns the document or resource this unit came from.
	SourceDocumentID() string

	// SourceURI returns the canonical location of the source document.
	// Used for citations and traceability.
	SourceURI() string

	// AsText returns a natural language representation of this unit
	// suitable for injection into an LLM prompt.
	AsText() string

	// Score returns the most recent relevance score.
	Score() float64

	// Scores returns the complete scoring history of this unit,
	// from initial retrieval through all reranking passes.
	Scores() []ScoreEntry

	// WithScore returns a copy of this unit with the given score
	// appended to the score history. The stage parameter identifies
	// which pipeline stage assigned the score.
	WithScore(stage string, score float64) RetrievedUnit

	// Metadata returns arbitrary key-value pairs attached to this unit.
	Metadata() map[string]any
}

// --- Chunk ---

// Chunk is the concrete RetrievedUnit for text-based retrieval strategies:
// vector search, sparse search, hybrid search, and their variants.
type Chunk struct {
	id          string
	sourceDocID string
	sourceURI   string
	text        string
	scores      []ScoreEntry
	metadata    map[string]any

	// Vector is the dense embedding of this chunk's text.
	// Nil until the embedder processes this chunk.
	Vector []float32

	// Position describes where in the source document this chunk came from.
	Position *Position

	// TokenCount is the number of tokens in this chunk's text,
	// computed by the Splitter at creation time. Zero if not computed.
	TokenCount int

	// ParentID, if non-empty, points to the larger parent chunk
	// this chunk was derived from (small-to-big retrieval).
	ParentID string

	// WindowText, if non-empty, is the surrounding sentence window.
	// When set, AsText returns this instead of the raw chunk text.
	WindowText string

	// ContextPrefix is a short LLM-generated sentence that situates
	// this chunk within its source document.
	ContextPrefix string
}

func (c *Chunk) ID() string               { return c.id }
func (c *Chunk) SourceDocumentID() string { return c.sourceDocID }
func (c *Chunk) SourceURI() string        { return c.sourceURI }
func (c *Chunk) Metadata() map[string]any { return c.metadata }

func (c *Chunk) Score() float64 {
	if len(c.scores) == 0 {
		return 0
	}
	return c.scores[len(c.scores)-1].Score
}

func (c *Chunk) Scores() []ScoreEntry {
	out := make([]ScoreEntry, len(c.scores))
	copy(out, c.scores)
	return out
}

func (c *Chunk) AsText() string {
	if c.WindowText != "" {
		return c.WindowText
	}
	if c.ContextPrefix != "" {
		return c.ContextPrefix + " " + c.text
	}
	return c.text
}

func (c *Chunk) WithScore(stage string, score float64) RetrievedUnit {
	cp := *c

	// Deep copy reference types to prevent shared mutation.
	if c.Vector != nil {
		cp.Vector = make([]float32, len(c.Vector))
		copy(cp.Vector, c.Vector)
	}
	if c.metadata != nil {
		cp.metadata = make(map[string]any, len(c.metadata))
		maps.Copy(cp.metadata, c.metadata)
	}
	if c.Position != nil {
		pos := *c.Position
		if c.Position.Section != nil {
			pos.Section = make([]string, len(c.Position.Section))
			copy(pos.Section, c.Position.Section)
		}
		cp.Position = &pos
	}

	cp.scores = make([]ScoreEntry, len(c.scores), len(c.scores)+1)
	copy(cp.scores, c.scores)
	cp.scores = append(cp.scores, ScoreEntry{Stage: stage, Score: score})

	return &cp
}

// Text returns the raw chunk text (before any window or context prefix).
func (c *Chunk) Text() string { return c.text }

// NewChunk constructs a Chunk with the required identity fields.
func NewChunk(id, sourceDocID, text string, metadata map[string]any) *Chunk {
	return &Chunk{
		id:          id,
		sourceDocID: sourceDocID,
		text:        text,
		metadata:    metadata,
	}
}

// NewChunkWithOptions constructs a Chunk with additional fields.
func NewChunkWithOptions(id, sourceDocID, text string, metadata map[string]any, sourceURI string, pos *Position, tokenCount int) *Chunk {
	return &Chunk{
		id:          id,
		sourceDocID: sourceDocID,
		text:        text,
		metadata:    metadata,
		sourceURI:   sourceURI,
		Position:    pos,
		TokenCount:  tokenCount,
	}
}

// --- Chunk JSON serialization ---

type chunkJSON struct {
	ID            string         `json:"id"`
	SourceDocID   string         `json:"source_doc_id"`
	SourceURI     string         `json:"source_uri,omitempty"`
	Text          string         `json:"text"`
	Vector        []float32      `json:"vector,omitempty"`
	Scores        []ScoreEntry   `json:"scores,omitempty"`
	Position      *Position      `json:"position,omitempty"`
	TokenCount    int            `json:"token_count,omitempty"`
	ParentID      string         `json:"parent_id,omitempty"`
	WindowText    string         `json:"window_text,omitempty"`
	ContextPrefix string         `json:"context_prefix,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

func (c *Chunk) MarshalJSON() ([]byte, error) {
	return json.Marshal(chunkJSON{
		ID:            c.id,
		SourceDocID:   c.sourceDocID,
		SourceURI:     c.sourceURI,
		Text:          c.text,
		Vector:        c.Vector,
		Scores:        c.scores,
		Position:      c.Position,
		TokenCount:    c.TokenCount,
		ParentID:      c.ParentID,
		WindowText:    c.WindowText,
		ContextPrefix: c.ContextPrefix,
		Metadata:      c.metadata,
	})
}

func (c *Chunk) UnmarshalJSON(data []byte) error {
	var j chunkJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return err
	}
	c.id = j.ID
	c.sourceDocID = j.SourceDocID
	c.sourceURI = j.SourceURI
	c.text = j.Text
	c.Vector = j.Vector
	c.scores = j.Scores
	c.Position = j.Position
	c.TokenCount = j.TokenCount
	c.ParentID = j.ParentID
	c.WindowText = j.WindowText
	c.ContextPrefix = j.ContextPrefix
	c.metadata = j.Metadata
	return nil
}

// --- GraphFact ---

// GraphFact is the concrete RetrievedUnit for graph retrieval.
// It represents a single fact extracted from the knowledge graph:
// a subject, a predicate (relationship type), and an object.
type GraphFact struct {
	id          string
	sourceDocID string
	sourceURI   string
	scores      []ScoreEntry
	metadata    map[string]any

	Subject   string
	Predicate string
	Object    string
}

func (g *GraphFact) ID() string               { return g.id }
func (g *GraphFact) SourceDocumentID() string { return g.sourceDocID }
func (g *GraphFact) SourceURI() string        { return g.sourceURI }
func (g *GraphFact) Metadata() map[string]any { return g.metadata }

func (g *GraphFact) Score() float64 {
	if len(g.scores) == 0 {
		return 0
	}
	return g.scores[len(g.scores)-1].Score
}

func (g *GraphFact) Scores() []ScoreEntry {
	out := make([]ScoreEntry, len(g.scores))
	copy(out, g.scores)
	return out
}

func (g *GraphFact) AsText() string {
	return fmt.Sprintf("%s is related to %s via %s.", g.Subject, g.Object, g.Predicate)
}

func (g *GraphFact) WithScore(stage string, score float64) RetrievedUnit {
	cp := *g
	if g.metadata != nil {
		cp.metadata = make(map[string]any, len(g.metadata))
		maps.Copy(cp.metadata, g.metadata)
	}
	cp.scores = make([]ScoreEntry, len(g.scores), len(g.scores)+1)
	copy(cp.scores, g.scores)
	cp.scores = append(cp.scores, ScoreEntry{Stage: stage, Score: score})
	return &cp
}

// NewGraphFact constructs a GraphFact with the required identity fields.
func NewGraphFact(id, sourceDocID, subject, predicate, object string, metadata map[string]any) *GraphFact {
	return &GraphFact{
		id:          id,
		sourceDocID: sourceDocID,
		Subject:     subject,
		Predicate:   predicate,
		Object:      object,
		metadata:    metadata,
	}
}

// --- GraphFact JSON serialization ---

type graphFactJSON struct {
	ID          string         `json:"id"`
	SourceDocID string         `json:"source_doc_id"`
	SourceURI   string         `json:"source_uri,omitempty"`
	Scores      []ScoreEntry   `json:"scores,omitempty"`
	Subject     string         `json:"subject"`
	Predicate   string         `json:"predicate"`
	Object      string         `json:"object"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

func (g *GraphFact) MarshalJSON() ([]byte, error) {
	return json.Marshal(graphFactJSON{
		ID:          g.id,
		SourceDocID: g.sourceDocID,
		SourceURI:   g.sourceURI,
		Scores:      g.scores,
		Subject:     g.Subject,
		Predicate:   g.Predicate,
		Object:      g.Object,
		Metadata:    g.metadata,
	})
}

func (g *GraphFact) UnmarshalJSON(data []byte) error {
	var j graphFactJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return err
	}
	g.id = j.ID
	g.sourceDocID = j.SourceDocID
	g.sourceURI = j.SourceURI
	g.scores = j.Scores
	g.Subject = j.Subject
	g.Predicate = j.Predicate
	g.Object = j.Object
	g.metadata = j.Metadata
	return nil
}

// --- StructuredRow ---

// StructuredRow is the concrete RetrievedUnit for structured retrieval —
// the case where the knowledge base is a database and the retriever
// executed a generated SQL query.
type StructuredRow struct {
	id          string
	sourceDocID string
	sourceURI   string
	scores      []ScoreEntry
	metadata    map[string]any

	// Columns preserves the relational structure of the row,
	// keyed by column name.
	Columns map[string]any
}

func (s *StructuredRow) ID() string               { return s.id }
func (s *StructuredRow) SourceDocumentID() string { return s.sourceDocID }
func (s *StructuredRow) SourceURI() string        { return s.sourceURI }
func (s *StructuredRow) Metadata() map[string]any { return s.metadata }

func (s *StructuredRow) Score() float64 {
	if len(s.scores) == 0 {
		return 0
	}
	return s.scores[len(s.scores)-1].Score
}

func (s *StructuredRow) Scores() []ScoreEntry {
	out := make([]ScoreEntry, len(s.scores))
	copy(out, s.scores)
	return out
}

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

func (s *StructuredRow) WithScore(stage string, score float64) RetrievedUnit {
	cp := *s
	if s.metadata != nil {
		cp.metadata = make(map[string]any, len(s.metadata))
		maps.Copy(cp.metadata, s.metadata)
	}
	if s.Columns != nil {
		cp.Columns = make(map[string]any, len(s.Columns))
		maps.Copy(cp.Columns, s.Columns)
	}
	cp.scores = make([]ScoreEntry, len(s.scores), len(s.scores)+1)
	copy(cp.scores, s.scores)
	cp.scores = append(cp.scores, ScoreEntry{Stage: stage, Score: score})
	return &cp
}

// NewStructuredRow constructs a StructuredRow with the required identity fields.
func NewStructuredRow(id, sourceDocID string, columns map[string]any, metadata map[string]any) *StructuredRow {
	return &StructuredRow{
		id:          id,
		sourceDocID: sourceDocID,
		Columns:     columns,
		metadata:    metadata,
	}
}

// --- StructuredRow JSON serialization ---

type structuredRowJSON struct {
	ID          string         `json:"id"`
	SourceDocID string         `json:"source_doc_id"`
	SourceURI   string         `json:"source_uri,omitempty"`
	Scores      []ScoreEntry   `json:"scores,omitempty"`
	Columns     map[string]any `json:"columns"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

func (s *StructuredRow) MarshalJSON() ([]byte, error) {
	return json.Marshal(structuredRowJSON{
		ID:          s.id,
		SourceDocID: s.sourceDocID,
		SourceURI:   s.sourceURI,
		Scores:      s.scores,
		Columns:     s.Columns,
		Metadata:    s.metadata,
	})
}

func (s *StructuredRow) UnmarshalJSON(data []byte) error {
	var j structuredRowJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return err
	}
	s.id = j.ID
	s.sourceDocID = j.SourceDocID
	s.sourceURI = j.SourceURI
	s.scores = j.Scores
	s.Columns = j.Columns
	s.metadata = j.Metadata
	return nil
}

// --- UnitEnvelope for cross-process serialization ---

// UnitType identifies the concrete type of a serialized RetrievedUnit.
type UnitType string

const (
	UnitTypeChunk         UnitType = "chunk"
	UnitTypeGraphFact     UnitType = "graph_fact"
	UnitTypeStructuredRow UnitType = "structured_row"
)

// UnitEnvelope is a serializable wrapper around a RetrievedUnit.
// It carries a type discriminator so that deserializers know
// which concrete type to decode into.
type UnitEnvelope struct {
	Type UnitType        `json:"type"`
	Data json.RawMessage `json:"data"`
}

// WrapUnit packs a RetrievedUnit into a serializable envelope.
func WrapUnit(u RetrievedUnit) (UnitEnvelope, error) {
	var unitType UnitType
	switch u.(type) {
	case *Chunk:
		unitType = UnitTypeChunk
	case *GraphFact:
		unitType = UnitTypeGraphFact
	case *StructuredRow:
		unitType = UnitTypeStructuredRow
	default:
		return UnitEnvelope{}, fmt.Errorf("dory: unknown RetrievedUnit type %T", u)
	}

	data, err := json.Marshal(u)
	if err != nil {
		return UnitEnvelope{}, fmt.Errorf("dory: marshal unit: %w", err)
	}

	return UnitEnvelope{Type: unitType, Data: data}, nil
}

// UnwrapUnit recovers a RetrievedUnit from an envelope.
func UnwrapUnit(e UnitEnvelope) (RetrievedUnit, error) {
	switch e.Type {
	case UnitTypeChunk:
		var c Chunk
		if err := json.Unmarshal(e.Data, &c); err != nil {
			return nil, fmt.Errorf("dory: unmarshal chunk: %w", err)
		}
		return &c, nil
	case UnitTypeGraphFact:
		var g GraphFact
		if err := json.Unmarshal(e.Data, &g); err != nil {
			return nil, fmt.Errorf("dory: unmarshal graph fact: %w", err)
		}
		return &g, nil
	case UnitTypeStructuredRow:
		var s StructuredRow
		if err := json.Unmarshal(e.Data, &s); err != nil {
			return nil, fmt.Errorf("dory: unmarshal structured row: %w", err)
		}
		return &s, nil
	default:
		return nil, fmt.Errorf("dory: unknown unit type %q", e.Type)
	}
}
