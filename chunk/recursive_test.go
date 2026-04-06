package chunk

import (
	"context"
	"testing"
)

func TestRecursive_Split(t *testing.T) {
	ctx := context.Background()

	t.Run("content shorter than size returns single chunk", func(t *testing.T) {
		s := NewRecursive(RecursiveConfig{Size: 100})
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

	t.Run("splits on paragraph boundaries", func(t *testing.T) {
		s := NewRecursive(RecursiveConfig{Size: 20})
		doc := mustDoc(t, "d1", "Hello world.\n\nSecond paragraph.")

		chunks, err := s.Split(ctx, doc)
		if err != nil {
			t.Fatal(err)
		}
		if len(chunks) != 2 {
			t.Fatalf("got %d chunks, want 2", len(chunks))
		}
		if chunks[0].AsText() != "Hello world.\n\n" {
			t.Errorf("chunk 0: got %q, want %q", chunks[0].AsText(), "Hello world.\n\n")
		}
		if chunks[1].AsText() != "Second paragraph." {
			t.Errorf("chunk 1: got %q, want %q", chunks[1].AsText(), "Second paragraph.")
		}
	})

	t.Run("falls back to newline separator", func(t *testing.T) {
		s := NewRecursive(RecursiveConfig{Size: 15})
		doc := mustDoc(t, "d1", "line one\nline two\nline three")

		chunks, err := s.Split(ctx, doc)
		if err != nil {
			t.Fatal(err)
		}
		if len(chunks) < 2 {
			t.Fatalf("got %d chunks, want at least 2", len(chunks))
		}
		// First chunk should respect the newline boundary.
		if chunks[0].AsText() != "line one\n" {
			t.Errorf("chunk 0: got %q, want %q", chunks[0].AsText(), "line one\n")
		}
	})

	t.Run("falls back to space separator", func(t *testing.T) {
		s := NewRecursive(RecursiveConfig{Size: 10})
		doc := mustDoc(t, "d1", "one two three four")

		chunks, err := s.Split(ctx, doc)
		if err != nil {
			t.Fatal(err)
		}
		if len(chunks) < 2 {
			t.Fatalf("got %d chunks, want at least 2", len(chunks))
		}
		// Each chunk should be <= 10 chars.
		for i, c := range chunks {
			if len(c.AsText()) > 10 {
				t.Errorf("chunk %d: length %d exceeds size 10: %q", i, len(c.AsText()), c.AsText())
			}
		}
	})

	t.Run("falls back to character split", func(t *testing.T) {
		s := NewRecursive(RecursiveConfig{Size: 5})
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

	t.Run("overlap between chunks", func(t *testing.T) {
		s := NewRecursive(RecursiveConfig{Size: 20, Overlap: 5})
		doc := mustDoc(t, "d1", "Hello world.\n\nSecond paragraph.")

		chunks, err := s.Split(ctx, doc)
		if err != nil {
			t.Fatal(err)
		}
		if len(chunks) < 2 {
			t.Fatalf("got %d chunks, want at least 2", len(chunks))
		}
		// Second chunk should overlap with the end of the first.
		if chunks[1].Position.StartByte >= chunks[0].Position.EndByte {
			t.Errorf("expected overlap: chunk 1 starts at %d, chunk 0 ends at %d",
				chunks[1].Position.StartByte, chunks[0].Position.EndByte)
		}
	})

	t.Run("empty content returns no chunks", func(t *testing.T) {
		s := NewRecursive(RecursiveConfig{Size: 100})
		doc := mustDoc(t, "d1", "")

		chunks, err := s.Split(ctx, doc)
		if err != nil {
			t.Fatal(err)
		}
		if len(chunks) != 0 {
			t.Fatalf("got %d chunks, want 0", len(chunks))
		}
	})

	t.Run("positions have correct byte offsets", func(t *testing.T) {
		s := NewRecursive(RecursiveConfig{Size: 20})
		text := "First paragraph.\n\nSecond paragraph."
		doc := mustDoc(t, "d1", text)

		chunks, err := s.Split(ctx, doc)
		if err != nil {
			t.Fatal(err)
		}
		for _, c := range chunks {
			if c.Position == nil {
				t.Fatal("expected position on chunk")
			}
			got := text[c.Position.StartByte:c.Position.EndByte]
			if got != c.AsText() {
				t.Errorf("position slice %q does not match chunk text %q", got, c.AsText())
			}
		}
	})

	t.Run("custom separators", func(t *testing.T) {
		s := NewRecursive(RecursiveConfig{
			Size:       8,
			Separators: []string{";", ""},
		})
		doc := mustDoc(t, "d1", "abc;defgh;ij")

		chunks, err := s.Split(ctx, doc)
		if err != nil {
			t.Fatal(err)
		}
		if len(chunks) < 2 {
			t.Fatalf("got %d chunks, want at least 2", len(chunks))
		}
		if chunks[0].AsText() != "abc;" {
			t.Errorf("chunk 0: got %q, want %q", chunks[0].AsText(), "abc;")
		}
	})

	t.Run("metadata is copied not shared", func(t *testing.T) {
		s := NewRecursive(RecursiveConfig{Size: 5})
		doc := mustDoc(t, "d1", "abcdefghij")

		chunks, err := s.Split(ctx, doc)
		if err != nil {
			t.Fatal(err)
		}
		if len(chunks) < 2 {
			t.Fatalf("need at least 2 chunks")
		}
		chunks[0].Metadata()["key"] = "changed"
		if chunks[1].Metadata()["key"] == "changed" {
			t.Error("metadata is shared between chunks, should be independent copies")
		}
	})

	t.Run("source document ID is propagated", func(t *testing.T) {
		s := NewRecursive(RecursiveConfig{Size: 100})
		doc := mustDoc(t, "my-doc", "hello")

		chunks, err := s.Split(ctx, doc)
		if err != nil {
			t.Fatal(err)
		}
		if chunks[0].SourceDocumentID() != "my-doc" {
			t.Errorf("got %q, want %q", chunks[0].SourceDocumentID(), "my-doc")
		}
	})
}
