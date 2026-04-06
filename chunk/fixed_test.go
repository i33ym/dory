package chunk

import (
	"context"
	"testing"

	"github.com/i33ym/dory"
)

func TestFixed_Split(t *testing.T) {
	ctx := context.Background()

	t.Run("single chunk when content fits within size", func(t *testing.T) {
		s := NewFixed(FixedConfig{Size: 100})
		doc := &dory.Document{ID: "d1", Content: "hello world"}

		chunks, err := s.Split(ctx, doc)
		if err != nil {
			t.Fatal(err)
		}
		if len(chunks) != 1 {
			t.Fatalf("got %d chunks, want 1", len(chunks))
		}
		if chunks[0].AsText() != "hello world" {
			t.Errorf("got %q, want %q", chunks[0].AsText(), "hello world")
		}
	})

	t.Run("splits content into equal sized chunks", func(t *testing.T) {
		s := NewFixed(FixedConfig{Size: 5})
		doc := &dory.Document{ID: "d1", Content: "abcdefghij"}

		chunks, err := s.Split(ctx, doc)
		if err != nil {
			t.Fatal(err)
		}
		if len(chunks) != 2 {
			t.Fatalf("got %d chunks, want 2", len(chunks))
		}
		if chunks[0].AsText() != "abcde" {
			t.Errorf("chunk 0: got %q, want %q", chunks[0].AsText(), "abcde")
		}
		if chunks[1].AsText() != "fghij" {
			t.Errorf("chunk 1: got %q, want %q", chunks[1].AsText(), "fghij")
		}
	})

	t.Run("last chunk can be shorter", func(t *testing.T) {
		s := NewFixed(FixedConfig{Size: 4})
		doc := &dory.Document{ID: "d1", Content: "abcdefg"}

		chunks, err := s.Split(ctx, doc)
		if err != nil {
			t.Fatal(err)
		}
		if len(chunks) != 2 {
			t.Fatalf("got %d chunks, want 2", len(chunks))
		}
		if chunks[0].AsText() != "abcd" {
			t.Errorf("chunk 0: got %q, want %q", chunks[0].AsText(), "abcd")
		}
		if chunks[1].AsText() != "efg" {
			t.Errorf("chunk 1: got %q, want %q", chunks[1].AsText(), "efg")
		}
	})

	t.Run("overlap between chunks", func(t *testing.T) {
		s := NewFixed(FixedConfig{Size: 6, Overlap: 2})
		doc := &dory.Document{ID: "d1", Content: "abcdefghijkl"}

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
		if chunks[2].AsText() != "ijkl" {
			t.Errorf("chunk 2: got %q, want %q", chunks[2].AsText(), "ijkl")
		}
	})

	t.Run("empty content returns no chunks", func(t *testing.T) {
		s := NewFixed(FixedConfig{Size: 10})
		doc := &dory.Document{ID: "d1", Content: ""}

		chunks, err := s.Split(ctx, doc)
		if err != nil {
			t.Fatal(err)
		}
		if len(chunks) != 0 {
			t.Fatalf("got %d chunks, want 0", len(chunks))
		}
	})

	t.Run("zero size defaults to 512", func(t *testing.T) {
		s := NewFixed(FixedConfig{Size: 0})
		content := make([]byte, 1024)
		for i := range content {
			content[i] = 'a'
		}
		doc := &dory.Document{ID: "d1", Content: string(content)}

		chunks, err := s.Split(ctx, doc)
		if err != nil {
			t.Fatal(err)
		}
		if len(chunks) != 2 {
			t.Fatalf("got %d chunks, want 2", len(chunks))
		}
		if len(chunks[0].AsText()) != 512 {
			t.Errorf("chunk 0 length: got %d, want 512", len(chunks[0].AsText()))
		}
	})

	t.Run("overlap >= size is reset to zero", func(t *testing.T) {
		s := NewFixed(FixedConfig{Size: 5, Overlap: 5})
		doc := &dory.Document{ID: "d1", Content: "abcdefghij"}

		chunks, err := s.Split(ctx, doc)
		if err != nil {
			t.Fatal(err)
		}
		if len(chunks) != 2 {
			t.Fatalf("got %d chunks, want 2", len(chunks))
		}
	})

	t.Run("chunk IDs are sequential", func(t *testing.T) {
		s := NewFixed(FixedConfig{Size: 3})
		doc := &dory.Document{ID: "doc", Content: "abcdefghi"}

		chunks, err := s.Split(ctx, doc)
		if err != nil {
			t.Fatal(err)
		}
		for i, c := range chunks {
			want := "doc-" + itoa(i)
			if c.ID() != want {
				t.Errorf("chunk %d: got ID %q, want %q", i, c.ID(), want)
			}
		}
	})

	t.Run("source document ID is propagated", func(t *testing.T) {
		s := NewFixed(FixedConfig{Size: 100})
		doc := &dory.Document{ID: "my-doc", Content: "hello"}

		chunks, err := s.Split(ctx, doc)
		if err != nil {
			t.Fatal(err)
		}
		if chunks[0].SourceDocumentID() != "my-doc" {
			t.Errorf("got %q, want %q", chunks[0].SourceDocumentID(), "my-doc")
		}
	})

	t.Run("metadata is copied not shared", func(t *testing.T) {
		s := NewFixed(FixedConfig{Size: 5})
		meta := map[string]any{"tenant": "acme"}
		doc := &dory.Document{ID: "d1", Content: "abcdefghij", Metadata: meta}

		chunks, err := s.Split(ctx, doc)
		if err != nil {
			t.Fatal(err)
		}

		// Mutating one chunk's metadata should not affect the other.
		chunks[0].Metadata()["tenant"] = "changed"
		if chunks[1].Metadata()["tenant"] != "acme" {
			t.Error("metadata is shared between chunks, should be independent copies")
		}
		// Original doc metadata should also be unaffected.
		if meta["tenant"] != "acme" {
			t.Error("chunk metadata mutation affected original document metadata")
		}
	})

	t.Run("nil metadata is handled", func(t *testing.T) {
		s := NewFixed(FixedConfig{Size: 100})
		doc := &dory.Document{ID: "d1", Content: "hello"}

		chunks, err := s.Split(ctx, doc)
		if err != nil {
			t.Fatal(err)
		}
		if chunks[0].Metadata() != nil {
			t.Errorf("expected nil metadata, got %v", chunks[0].Metadata())
		}
	})
}
