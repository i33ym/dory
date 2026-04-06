package chunk

import (
	"context"
	"strings"

	"github.com/i33ym/dory"
	"github.com/i33ym/dory/internal/similarity"
)

// SemanticConfig configures the semantic boundary chunking strategy.
type SemanticConfig struct {
	// MaxSize is the maximum number of characters per chunk.
	MaxSize int

	// SimThreshold is the cosine similarity threshold below which a new
	// chunk boundary is created. Defaults to 0.5.
	SimThreshold float64

	// Embedder computes vector embeddings for sentences.
	Embedder dory.Embedder
}

// Semantic splits documents at semantic boundaries by detecting topic shifts
// using sentence embeddings. Consecutive sentences with high cosine similarity
// are grouped together; when similarity drops below SimThreshold, a new chunk
// begins.
type Semantic struct {
	config SemanticConfig
}

// NewSemantic creates a new semantic boundary splitter.
func NewSemantic(config SemanticConfig) *Semantic {
	return &Semantic{config: config}
}

// Split implements the [dory.Splitter] interface.
func (s *Semantic) Split(ctx context.Context, doc *dory.Document) ([]*dory.Chunk, error) {
	content, err := doc.Content().Text()
	if err != nil {
		return nil, err
	}

	if len(content) == 0 {
		return nil, nil
	}

	maxSize := s.config.MaxSize
	if maxSize <= 0 {
		maxSize = 512
	}
	threshold := s.config.SimThreshold
	if threshold == 0 {
		threshold = 0.5
	}

	sentences := splitSentences(content)
	if len(sentences) == 0 {
		return nil, nil
	}

	// Embed all sentences.
	embeddings := make([][]float32, len(sentences))
	for i, sent := range sentences {
		vec, err := s.config.Embedder.Embed(ctx, sent)
		if err != nil {
			return nil, err
		}
		embeddings[i] = vec
	}

	// Group sentences into segments based on similarity drops.
	type segment struct {
		sentences []string
		start     int // index of first sentence
	}

	var segments []segment
	cur := segment{sentences: []string{sentences[0]}, start: 0}

	for i := 1; i < len(sentences); i++ {
		sim := similarity.Cosine(embeddings[i-1], embeddings[i])
		if sim < threshold {
			// Topic shift: flush current segment and start new one.
			segments = append(segments, cur)
			cur = segment{sentences: []string{sentences[i]}, start: i}
		} else {
			cur.sentences = append(cur.sentences, sentences[i])
		}
	}
	segments = append(segments, cur)

	// Merge small segments if they fit under MaxSize.
	var merged []segment
	for _, seg := range segments {
		text := strings.Join(seg.sentences, "")
		if len(merged) > 0 {
			prev := &merged[len(merged)-1]
			prevText := strings.Join(prev.sentences, "")
			if len(prevText)+len(text) <= maxSize {
				prev.sentences = append(prev.sentences, seg.sentences...)
				continue
			}
		}
		merged = append(merged, seg)
	}

	// Build chunks from merged segments.
	var chunks []*dory.Chunk
	offset := 0
	for _, seg := range merged {
		// Calculate the byte offset for this segment.
		startByte := 0
		for i := 0; i < seg.start; i++ {
			startByte += len(sentences[i])
		}
		text := strings.Join(seg.sentences, "")
		endByte := startByte + len(text)

		chunk := dory.NewChunkWithOptions(
			doc.ID()+"-"+itoa(len(chunks)),
			doc.ID(),
			text,
			copyMeta(doc.Metadata()),
			doc.SourceURI(),
			&dory.Position{StartByte: startByte, EndByte: endByte},
			0,
		)
		chunks = append(chunks, chunk)
		offset = endByte
	}
	_ = offset

	return chunks, nil
}
