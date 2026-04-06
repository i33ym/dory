package chunk

import (
	"context"
	"maps"

	"github.com/i33ym/dory"
)

// LateConfig configures the late chunking strategy.
type LateConfig struct {
	// Size is the maximum number of characters per chunk.
	Size int

	// Overlap is the number of characters shared between consecutive chunks.
	Overlap int

	// Embedder computes vector embeddings.
	Embedder dory.Embedder
}

// Late splits documents into fixed-size chunks and marks them for late
// interaction. In late chunking the document is conceptually embedded as a
// whole and each chunk's embedding is derived from the corresponding portion.
// This simplified implementation embeds each chunk individually and sets the
// metadata key "late_chunking" to true.
type Late struct {
	config LateConfig
}

// NewLate creates a new late chunking splitter.
func NewLate(config LateConfig) *Late {
	return &Late{config: config}
}

// Split implements the [dory.Splitter] interface.
func (l *Late) Split(ctx context.Context, doc *dory.Document) ([]*dory.Chunk, error) {
	content, err := doc.Content().Text()
	if err != nil {
		return nil, err
	}

	size := l.config.Size
	overlap := l.config.Overlap

	if size <= 0 {
		size = 512
	}
	if overlap < 0 || overlap >= size {
		overlap = 0
	}

	if len(content) == 0 {
		return nil, nil
	}

	var chunks []*dory.Chunk
	step := size - overlap
	for i := 0; i < len(content); i += step {
		end := min(i+size, len(content))
		text := content[i:end]

		meta := copyMeta(doc.Metadata())
		if meta == nil {
			meta = make(map[string]any)
		}
		maps.Copy(meta, map[string]any{"late_chunking": true})

		// Embed the chunk text.
		vec, err := l.config.Embedder.Embed(ctx, text)
		if err != nil {
			return nil, err
		}

		chunk := dory.NewChunkWithOptions(
			doc.ID()+"-"+itoa(len(chunks)),
			doc.ID(),
			text,
			meta,
			doc.SourceURI(),
			&dory.Position{StartByte: i, EndByte: end},
			0,
		)
		chunk.Vector = vec
		chunks = append(chunks, chunk)

		if end == len(content) {
			break
		}
	}
	return chunks, nil
}
