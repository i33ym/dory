package chunk

import (
	"context"
	"testing"

	"github.com/i33ym/dory"
)

func mustDoc(t *testing.T, id, text string, opts ...dory.DocumentOption) *dory.Document {
	t.Helper()
	doc, err := dory.NewDocument(id, dory.TextContent(text, ""), opts...)
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

func TestFixed_Split(t *testing.T) {
	ctx := context.Background()

	t.Run("single chunk when content fits within size", func(t *testing.T) {
		s := NewFixed(FixedConfig{Size: 100})
		doc := mustDoc(t, "d1", "hello world")

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
		doc := mustDoc(t, "d1", "abcdefghij")

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
		doc := mustDoc(t, "d1", "abcdefg")

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
		if chunks[2].AsText() != "ijkl" {
			t.Errorf("chunk 2: got %q, want %q", chunks[2].AsText(), "ijkl")
		}
	})

	t.Run("empty content returns no chunks", func(t *testing.T) {
		s := NewFixed(FixedConfig{Size: 10})
		doc := mustDoc(t, "d1", "")

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
		doc := mustDoc(t, "d1", string(content))

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
		doc := mustDoc(t, "d1", "abcdefghij")

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
		doc := mustDoc(t, "doc", "abcdefghi")

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
		doc := mustDoc(t, "my-doc", "hello")

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
		doc := mustDoc(t, "d1", "abcdefghij", dory.WithMetadata("tenant", "acme"))

		chunks, err := s.Split(ctx, doc)
		if err != nil {
			t.Fatal(err)
		}

		// Mutating one chunk's metadata should not affect the other.
		chunks[0].Metadata()["tenant"] = "changed"
		if chunks[1].Metadata()["tenant"] != "acme" {
			t.Error("metadata is shared between chunks, should be independent copies")
		}
	})

	t.Run("nil metadata is handled", func(t *testing.T) {
		s := NewFixed(FixedConfig{Size: 100})
		doc := mustDoc(t, "d1", "hello")

		chunks, err := s.Split(ctx, doc)
		if err != nil {
			t.Fatal(err)
		}
		// NewDocument always initializes metadata, so it won't be nil.
		// But chunk metadata should exist.
		if chunks[0].Metadata() == nil {
			t.Errorf("expected non-nil metadata")
		}
	})

	t.Run("position is set on each chunk", func(t *testing.T) {
		s := NewFixed(FixedConfig{Size: 5})
		doc := mustDoc(t, "d1", "abcdefghij")

		chunks, err := s.Split(ctx, doc)
		if err != nil {
			t.Fatal(err)
		}
		if chunks[0].Position == nil {
			t.Fatal("expected position on chunk 0")
		}
		if chunks[0].Position.StartByte != 0 || chunks[0].Position.EndByte != 5 {
			t.Errorf("chunk 0 position: got %d-%d, want 0-5",
				chunks[0].Position.StartByte, chunks[0].Position.EndByte)
		}
		if chunks[1].Position.StartByte != 5 || chunks[1].Position.EndByte != 10 {
			t.Errorf("chunk 1 position: got %d-%d, want 5-10",
				chunks[1].Position.StartByte, chunks[1].Position.EndByte)
		}
	})

	t.Run("source URI is propagated", func(t *testing.T) {
		s := NewFixed(FixedConfig{Size: 100})
		doc := mustDoc(t, "d1", "hello", dory.WithSourceURI("s3://bucket/file.txt"))

		chunks, err := s.Split(ctx, doc)
		if err != nil {
			t.Fatal(err)
		}
		if chunks[0].SourceURI() != "s3://bucket/file.txt" {
			t.Errorf("got %q, want %q", chunks[0].SourceURI(), "s3://bucket/file.txt")
		}
	})
}
