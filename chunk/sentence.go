package chunk

import (
	"context"
	"strings"

	"github.com/i33ym/dory"
)

// sentenceEndings defines the punctuation+delimiter pairs that mark sentence
// boundaries. The punctuation is kept with the preceding sentence.
var sentenceEndings = []string{". ", "! ", "? ", ".\n", "!\n", "?\n"}

// SentenceConfig configures the sentence-aware splitter.
type SentenceConfig struct {
	// Size is the maximum number of characters per chunk.
	Size int

	// Overlap is the number of sentences carried over between consecutive chunks.
	Overlap int
}

// Sentence splits documents into sentence-aware chunks. Text is first
// split into sentences, then sentences are grouped into chunks up to
// Size characters. Overlap controls how many sentences are shared
// between consecutive chunks.
type Sentence struct {
	config SentenceConfig
}

// NewSentence creates a new sentence-aware splitter.
func NewSentence(config SentenceConfig) *Sentence {
	return &Sentence{config: config}
}

// Split implements the [dory.Splitter] interface.
func (s *Sentence) Split(_ context.Context, doc *dory.Document) ([]*dory.Chunk, error) {
	content, err := doc.Content().Text()
	if err != nil {
		return nil, err
	}

	size := s.config.Size
	overlap := s.config.Overlap

	if size <= 0 {
		size = 512
	}
	if overlap < 0 {
		overlap = 0
	}

	if len(content) == 0 {
		return nil, nil
	}

	sentences := splitSentences(content)
	if len(sentences) == 0 {
		return nil, nil
	}

	var chunks []*dory.Chunk
	i := 0
	for i < len(sentences) {
		// Determine which sentences go into this chunk.
		group := []string{sentences[i]}
		groupLen := len(sentences[i])
		j := i + 1
		for j < len(sentences) {
			if groupLen+len(sentences[j]) > size {
				break
			}
			group = append(group, sentences[j])
			groupLen += len(sentences[j])
			j++
		}

		// If a single sentence exceeds size, we still include it as one chunk.
		text := strings.Join(group, "")

		// Calculate byte offset: find where this text appears in content.
		startByte := findOffset(content, sentences, i)
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

		// Advance, carrying overlap sentences back.
		sentencesUsed := j - i
		if overlap > 0 && overlap < sentencesUsed {
			i = j - overlap
		} else {
			i = j
		}
	}
	return chunks, nil
}

// findOffset calculates the byte offset of sentence at index idx by summing
// the lengths of all preceding sentences.
func findOffset(content string, sentences []string, idx int) int {
	_ = content // kept for clarity; offset is just the sum of prior sentence lengths
	offset := 0
	for i := range idx {
		offset += len(sentences[i])
	}
	return offset
}

// splitSentences splits text into sentences, keeping the terminating
// punctuation and delimiter attached to each sentence.
func splitSentences(text string) []string {
	var sentences []string
	remaining := text

	for len(remaining) > 0 {
		// Find the earliest sentence ending.
		bestIdx := -1
		bestLen := 0
		for _, ending := range sentenceEndings {
			idx := strings.Index(remaining, ending)
			if idx >= 0 && (bestIdx < 0 || idx < bestIdx) {
				bestIdx = idx
				bestLen = len(ending)
			}
		}

		if bestIdx < 0 {
			// No more sentence endings; the rest is one sentence.
			sentences = append(sentences, remaining)
			break
		}

		// Include the ending with the sentence.
		sentence := remaining[:bestIdx+bestLen]
		sentences = append(sentences, sentence)
		remaining = remaining[bestIdx+bestLen:]
	}
	return sentences
}
