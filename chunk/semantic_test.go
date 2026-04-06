package chunk

import (
	"context"
	"testing"
)

// fakeEmbedder returns pre-configured vectors for each call.
type fakeEmbedder struct {
	vectors [][]float32
	idx     int
}

func (f *fakeEmbedder) Embed(_ context.Context, _ string) ([]float32, error) {
	v := f.vectors[f.idx%len(f.vectors)]
	f.idx++
	return v, nil
}

func (f *fakeEmbedder) EmbedBatch(_ context.Context, texts []string) ([][]float32, error) {
	var result [][]float32
	for range texts {
		v := f.vectors[f.idx%len(f.vectors)]
		f.idx++
		result = append(result, v)
	}
	return result, nil
}

func (f *fakeEmbedder) Dimensions() int {
	if len(f.vectors) > 0 {
		return len(f.vectors[0])
	}
	return 0
}

func TestSemantic_Split(t *testing.T) {
	ctx := context.Background()

	t.Run("topic shift creates new chunk", func(t *testing.T) {
		// Three sentences. First two are similar (same vector), third is different.
		doc := mustDoc(t, "d1", "The cat sat. The cat slept. The stock rose. ")

		embedder := &fakeEmbedder{
			vectors: [][]float32{
				{1, 0, 0}, // sentence 1
				{1, 0, 0}, // sentence 2 - same topic
				{0, 1, 0}, // sentence 3 - different topic (cosine sim = 0)
			},
		}

		s := NewSemantic(SemanticConfig{
			MaxSize:      30, // small enough to prevent merging across topic shift
			SimThreshold: 0.5,
			Embedder:     embedder,
		})

		chunks, err := s.Split(ctx, doc)
		if err != nil {
			t.Fatal(err)
		}

		if len(chunks) != 2 {
			t.Fatalf("got %d chunks, want 2", len(chunks))
		}

		if chunks[0].AsText() != "The cat sat. The cat slept. " {
			t.Errorf("chunk 0: got %q", chunks[0].AsText())
		}
		if chunks[1].AsText() != "The stock rose. " {
			t.Errorf("chunk 1: got %q", chunks[1].AsText())
		}
	})

	t.Run("all similar sentences produce one chunk", func(t *testing.T) {
		doc := mustDoc(t, "d1", "A is good. B is good. C is good. ")

		embedder := &fakeEmbedder{
			vectors: [][]float32{
				{1, 1, 0},
				{1, 1, 0},
				{1, 1, 0},
			},
		}

		s := NewSemantic(SemanticConfig{
			MaxSize:      1000,
			SimThreshold: 0.5,
			Embedder:     embedder,
		})

		chunks, err := s.Split(ctx, doc)
		if err != nil {
			t.Fatal(err)
		}

		if len(chunks) != 1 {
			t.Fatalf("got %d chunks, want 1", len(chunks))
		}
	})

	t.Run("empty content returns nil", func(t *testing.T) {
		doc := mustDoc(t, "d1", "")
		embedder := &fakeEmbedder{}
		s := NewSemantic(SemanticConfig{Embedder: embedder})

		chunks, err := s.Split(ctx, doc)
		if err != nil {
			t.Fatal(err)
		}
		if chunks != nil {
			t.Fatalf("got %d chunks, want nil", len(chunks))
		}
	})

	t.Run("small segments are merged under MaxSize", func(t *testing.T) {
		// Four sentences, each pair is different topic but total is under MaxSize.
		doc := mustDoc(t, "d1", "A. B. C. D. ")

		embedder := &fakeEmbedder{
			vectors: [][]float32{
				{1, 0, 0}, // A
				{0, 1, 0}, // B - shift
				{0, 1, 0}, // C - same as B
				{0, 0, 1}, // D - shift
			},
		}

		s := NewSemantic(SemanticConfig{
			MaxSize:      1000, // large enough to merge everything
			SimThreshold: 0.5,
			Embedder:     embedder,
		})

		chunks, err := s.Split(ctx, doc)
		if err != nil {
			t.Fatal(err)
		}

		// All segments should merge into one chunk since total is under MaxSize.
		if len(chunks) != 1 {
			t.Fatalf("got %d chunks, want 1", len(chunks))
		}
	})

	t.Run("position is set correctly", func(t *testing.T) {
		doc := mustDoc(t, "d1", "Hello world. Goodbye world. ")

		embedder := &fakeEmbedder{
			vectors: [][]float32{
				{1, 0},
				{0, 1}, // shift
			},
		}

		s := NewSemantic(SemanticConfig{
			MaxSize:      10, // small enough to prevent merging
			SimThreshold: 0.5,
			Embedder:     embedder,
		})

		chunks, err := s.Split(ctx, doc)
		if err != nil {
			t.Fatal(err)
		}

		if len(chunks) < 2 {
			t.Fatalf("got %d chunks, want at least 2", len(chunks))
		}

		if chunks[0].Position.StartByte != 0 {
			t.Errorf("chunk 0 StartByte: got %d, want 0", chunks[0].Position.StartByte)
		}
	})
}
