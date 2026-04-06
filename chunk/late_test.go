package chunk

import (
	"context"
	"testing"
)

func TestLate_Split(t *testing.T) {
	ctx := context.Background()

	t.Run("chunks have late_chunking metadata", func(t *testing.T) {
		embedder := &fakeEmbedder{
			vectors: [][]float32{
				{0.1, 0.2, 0.3},
				{0.4, 0.5, 0.6},
			},
		}

		s := NewLate(LateConfig{Size: 5, Embedder: embedder})
		doc := mustDoc(t, "d1", "abcdefghij")

		chunks, err := s.Split(ctx, doc)
		if err != nil {
			t.Fatal(err)
		}

		if len(chunks) != 2 {
			t.Fatalf("got %d chunks, want 2", len(chunks))
		}

		for i, c := range chunks {
			val, ok := c.Metadata()["late_chunking"]
			if !ok {
				t.Errorf("chunk %d: missing late_chunking metadata", i)
				continue
			}
			if val != true {
				t.Errorf("chunk %d: late_chunking = %v, want true", i, val)
			}
		}
	})

	t.Run("chunks have embeddings set", func(t *testing.T) {
		embedder := &fakeEmbedder{
			vectors: [][]float32{
				{1, 2, 3},
			},
		}

		s := NewLate(LateConfig{Size: 100, Embedder: embedder})
		doc := mustDoc(t, "d1", "hello")

		chunks, err := s.Split(ctx, doc)
		if err != nil {
			t.Fatal(err)
		}

		if len(chunks) != 1 {
			t.Fatalf("got %d chunks, want 1", len(chunks))
		}
		if chunks[0].Vector == nil {
			t.Fatal("expected Vector to be set")
		}
		if chunks[0].Vector[0] != 1 || chunks[0].Vector[1] != 2 || chunks[0].Vector[2] != 3 {
			t.Errorf("got vector %v, want [1 2 3]", chunks[0].Vector)
		}
	})

	t.Run("overlap works", func(t *testing.T) {
		embedder := &fakeEmbedder{
			vectors: [][]float32{
				{1, 0},
				{0, 1},
				{1, 1},
			},
		}

		s := NewLate(LateConfig{Size: 6, Overlap: 2, Embedder: embedder})
		doc := mustDoc(t, "d1", "abcdefghijkl")

		chunks, err := s.Split(ctx, doc)
		if err != nil {
			t.Fatal(err)
		}

		if len(chunks) != 3 {
			t.Fatalf("got %d chunks, want 3", len(chunks))
		}
		if chunks[0].AsText() != "abcdef" {
			t.Errorf("chunk 0: got %q, want %q", chunks[0].AsText(), "abcdef")
		}
		if chunks[1].AsText() != "efghij" {
			t.Errorf("chunk 1: got %q, want %q", chunks[1].AsText(), "efghij")
		}
	})

	t.Run("empty content returns nil", func(t *testing.T) {
		embedder := &fakeEmbedder{}
		s := NewLate(LateConfig{Embedder: embedder})
		doc := mustDoc(t, "d1", "")

		chunks, err := s.Split(ctx, doc)
		if err != nil {
			t.Fatal(err)
		}
		if chunks != nil {
			t.Fatalf("got %d chunks, want nil", len(chunks))
		}
	})
}
