package chunk

import (
	"context"
	"strings"

	"github.com/i33ym/dory"
)

// defaultSeparators is the hierarchy of separators used by the recursive splitter.
var defaultSeparators = []string{"\n\n", "\n", ". ", " ", ""}

// RecursiveConfig configures the recursive character text splitter.
type RecursiveConfig struct {
	// Size is the maximum number of characters per chunk.
	Size int

	// Overlap is the number of characters shared between consecutive chunks.
	Overlap int

	// Separators overrides the default separator hierarchy.
	// If nil, defaults to: "\n\n", "\n", ". ", " ", "".
	Separators []string
}

// Recursive splits documents by recursively trying a hierarchy of separators,
// from the most semantically meaningful (paragraph breaks) down to individual
// characters. This preserves natural text boundaries where possible.
type Recursive struct {
	config RecursiveConfig
}

// NewRecursive creates a new recursive character text splitter.
func NewRecursive(config RecursiveConfig) *Recursive {
	return &Recursive{config: config}
}

// Split implements the [dory.Splitter] interface.
func (r *Recursive) Split(_ context.Context, doc *dory.Document) ([]*dory.Chunk, error) {
	content, err := doc.Content().Text()
	if err != nil {
		return nil, err
	}

	size := r.config.Size
	overlap := r.config.Overlap
	separators := r.config.Separators

	if size <= 0 {
		size = 512
	}
	if overlap < 0 || overlap >= size {
		overlap = 0
	}
	if len(separators) == 0 {
		separators = defaultSeparators
	}

	if len(content) == 0 {
		return nil, nil
	}

	// Split recursively into raw text pieces.
	pieces := recursiveSplit(content, separators, size)

	// Apply overlap and build chunks with correct byte offsets.
	var chunks []*dory.Chunk
	offset := 0
	for i, piece := range pieces {
		if i > 0 && overlap > 0 {
			// Find the start of this piece in the original content.
			prevEnd := offset
			// Back up by overlap characters, but don't go before the previous chunk's start.
			overlapStart := max(prevEnd-overlap, 0)
			text := content[overlapStart : offset+len(piece)]
			chunk := dory.NewChunkWithOptions(
				doc.ID()+"-"+itoa(len(chunks)),
				doc.ID(),
				text,
				copyMeta(doc.Metadata()),
				doc.SourceURI(),
				&dory.Position{StartByte: overlapStart, EndByte: offset + len(piece)},
				0,
			)
			chunks = append(chunks, chunk)
		} else {
			chunk := dory.NewChunkWithOptions(
				doc.ID()+"-"+itoa(len(chunks)),
				doc.ID(),
				piece,
				copyMeta(doc.Metadata()),
				doc.SourceURI(),
				&dory.Position{StartByte: offset, EndByte: offset + len(piece)},
				0,
			)
			chunks = append(chunks, chunk)
		}
		offset += len(piece)
	}
	return chunks, nil
}

// recursiveSplit splits text using the separator hierarchy. If a piece exceeds
// maxSize, it is further split using the next separator in the hierarchy.
func recursiveSplit(text string, separators []string, maxSize int) []string {
	if len(text) <= maxSize {
		return []string{text}
	}
	if len(separators) == 0 {
		// Should not happen since "" is the last resort, but be safe.
		return []string{text}
	}

	sep := separators[0]
	remaining := separators[1:]

	var parts []string
	if sep == "" {
		// Character-level split: break into maxSize pieces.
		for i := 0; i < len(text); i += maxSize {
			end := min(i+maxSize, len(text))
			parts = append(parts, text[i:end])
		}
		return parts
	}

	segments := splitKeepSep(text, sep)
	if len(segments) <= 1 {
		// Separator not found; try next separator.
		return recursiveSplit(text, remaining, maxSize)
	}

	// Merge segments into pieces up to maxSize, then recursively split
	// any piece that still exceeds maxSize.
	var result []string
	current := ""
	for _, seg := range segments {
		candidate := current + seg
		if len(candidate) > maxSize && current != "" {
			// Flush current.
			result = append(result, recursiveSplit(current, remaining, maxSize)...)
			current = seg
		} else {
			current = candidate
		}
	}
	if current != "" {
		result = append(result, recursiveSplit(current, remaining, maxSize)...)
	}
	return result
}

// splitKeepSep splits text by sep but keeps the separator attached to the
// segment that precedes it. For example, splitting "a\n\nb" by "\n\n"
// gives ["a\n\n", "b"].
func splitKeepSep(text, sep string) []string {
	raw := strings.Split(text, sep)
	if len(raw) <= 1 {
		return raw
	}
	result := make([]string, 0, len(raw))
	for i, part := range raw {
		if i < len(raw)-1 {
			result = append(result, part+sep)
		} else {
			if part != "" {
				result = append(result, part)
			}
		}
	}
	return result
}
