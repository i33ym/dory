// Package chunk provides text splitting strategies for Dory.
// Each strategy implements the [dory.Splitter] interface.
package chunk

import (
	"context"
	"maps"

	"github.com/i33ym/dory"
)

// FixedConfig configures the fixed-size chunking strategy.
type FixedConfig struct {
	// Size is the maximum number of characters per chunk.
	Size int

	// Overlap is the number of characters shared between consecutive chunks.
	Overlap int
}

// Fixed splits documents into chunks of a fixed character size with optional overlap.
type Fixed struct {
	config FixedConfig
}

// NewFixed creates a new fixed-size splitter with the given configuration.
func NewFixed(config FixedConfig) *Fixed {
	return &Fixed{config: config}
}

// Split implements the [dory.Splitter] interface.
func (f *Fixed) Split(_ context.Context, doc *dory.Document) ([]*dory.Chunk, error) {
	content, err := doc.Content().Text()
	if err != nil {
		return nil, err
	}

	size := f.config.Size
	overlap := f.config.Overlap

	if size <= 0 {
		size = 512
	}
	if overlap < 0 || overlap >= size {
		overlap = 0
	}

	var chunks []*dory.Chunk
	step := size - overlap
	for i := 0; i < len(content); i += step {
		end := min(i+size, len(content))
		chunk := dory.NewChunkWithOptions(
			doc.ID()+"-"+itoa(len(chunks)),
			doc.ID(),
			content[i:end],
			copyMeta(doc.Metadata()),
			doc.SourceURI(),
			&dory.Position{StartByte: i, EndByte: end},
			0,
		)
		chunks = append(chunks, chunk)
		if end == len(content) {
			break
		}
	}
	return chunks, nil
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := ""
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}
	return digits
}

func copyMeta(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	cp := make(map[string]any, len(m))
	maps.Copy(cp, m)
	return cp
}
