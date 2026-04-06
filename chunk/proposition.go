package chunk

import (
	"context"

	"github.com/i33ym/dory"
)

// PropositionConfig configures the proposition extraction strategy.
type PropositionConfig struct {
	// ExtractFunc extracts atomic propositions (self-contained factual
	// statements) from a text block. Typically backed by an LLM.
	ExtractFunc func(ctx context.Context, text string) ([]string, error)
}

// Proposition splits documents by extracting atomic propositions using an LLM.
// Each proposition becomes its own chunk.
type Proposition struct {
	config PropositionConfig
}

// NewProposition creates a new proposition extraction splitter.
func NewProposition(config PropositionConfig) *Proposition {
	return &Proposition{config: config}
}

// Split implements the [dory.Splitter] interface.
func (p *Proposition) Split(ctx context.Context, doc *dory.Document) ([]*dory.Chunk, error) {
	content, err := doc.Content().Text()
	if err != nil {
		return nil, err
	}

	if len(content) == 0 {
		return nil, nil
	}

	propositions, err := p.config.ExtractFunc(ctx, content)
	if err != nil {
		return nil, err
	}

	var chunks []*dory.Chunk
	for _, prop := range propositions {
		chunk := dory.NewChunkWithOptions(
			doc.ID()+"-"+itoa(len(chunks)),
			doc.ID(),
			prop,
			copyMeta(doc.Metadata()),
			doc.SourceURI(),
			&dory.Position{StartByte: 0, EndByte: len(content)},
			0,
		)
		chunks = append(chunks, chunk)
	}
	return chunks, nil
}
