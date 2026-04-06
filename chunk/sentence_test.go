package chunk

import (
	"context"
	"testing"

	"github.com/i33ym/dory"
)

func TestSentence_Split(t *testing.T) {
	ctx := context.Background()

	t.Run("single sentence under size", func(t *testing.T) {
		s := NewSentence(SentenceConfig{Size: 100})
		doc := mustDoc(t, "d1", "Hello world.")

		chunks, err := s.Split(ctx, doc)
		if err != nil {
			t.Fatal(err)
		}
		if len(chunks) != 1 {
			t.Fatalf("got %d chunks, want 1", len(chunks))
		}
		if chunks[0].AsText() != "Hello world." {
			t.Errorf("got %q, want %q", chunks[0].AsText(), "Hello world.")
		}
	})

	t.Run("groups sentences up to size", func(t *testing.T) {
		s := NewSentence(SentenceConfig{Size: 20})
		doc := mustDoc(t, "d1", "First. Second. Third. Fourth.")

		chunks, err := s.Split(ctx, doc)
		if err != nil {
			t.Fatal(err)
		}
		if len(chunks) < 2 {
			t.Fatalf("got %d chunks, want at least 2", len(chunks))
		}
		// Each chunk (except possibly a single oversized sentence) should be <= size.
		for i, c := range chunks {
			// Only check non-single-sentence chunks.
			if len(c.AsText()) > 20 {
				t.Errorf("chunk %d: length %d exceeds size 20: %q", i, len(c.AsText()), c.AsText())
			}
		}
	})

	t.Run("sentence overlap", func(t *testing.T) {
		s := NewSentence(SentenceConfig{Size: 25, Overlap: 1})
		doc := mustDoc(t, "d1", "AAA. BBB. CCC. DDD.")

		chunks, err := s.Split(ctx, doc)
		if err != nil {
			t.Fatal(err)
		}
		if len(chunks) < 2 {
			t.Fatalf("got %d chunks, want at least 2", len(chunks))
		}
		// With overlap=1, the last sentence of chunk N should appear at
		// the start of chunk N+1.
		for i := 0; i < len(chunks)-1; i++ {
			// The chunks should overlap in the content.
			if chunks[i+1].Position.StartByte >= chunks[i].Position.EndByte {
				t.Errorf("expected overlap between chunk %d (end=%d) and chunk %d (start=%d)",
					i, chunks[i].Position.EndByte, i+1, chunks[i+1].Position.StartByte)
			}
		}
	})

	t.Run("keeps punctuation with sentence", func(t *testing.T) {
		s := NewSentence(SentenceConfig{Size: 20})
		doc := mustDoc(t, "d1", "Hello! World? Yes.")

		chunks, err := s.Split(ctx, doc)
		if err != nil {
			t.Fatal(err)
		}
		// First chunk should include "Hello! "
		if len(chunks) == 0 {
			t.Fatal("expected at least 1 chunk")
		}
		text := chunks[0].AsText()
		if len(text) < 7 || text[:7] != "Hello! " {
			t.Errorf("expected chunk to start with %q, got %q", "Hello! ", text)
		}
	})

	t.Run("empty content returns no chunks", func(t *testing.T) {
		s := NewSentence(SentenceConfig{Size: 100})
		doc := mustDoc(t, "d1", "")

		chunks, err := s.Split(ctx, doc)
		if err != nil {
			t.Fatal(err)
		}
		if len(chunks) != 0 {
			t.Fatalf("got %d chunks, want 0", len(chunks))
		}
	})

	t.Run("single oversized sentence is kept as one chunk", func(t *testing.T) {
		s := NewSentence(SentenceConfig{Size: 5})
		doc := mustDoc(t, "d1", "This is a very long sentence without ending")

		chunks, err := s.Split(ctx, doc)
		if err != nil {
			t.Fatal(err)
		}
		if len(chunks) != 1 {
			t.Fatalf("got %d chunks, want 1", len(chunks))
		}
		if chunks[0].AsText() != "This is a very long sentence without ending" {
			t.Errorf("got %q", chunks[0].AsText())
		}
	})

	t.Run("positions have correct byte offsets", func(t *testing.T) {
		s := NewSentence(SentenceConfig{Size: 30})
		text := "First sentence. Second one. Third here."
		doc := mustDoc(t, "d1", text)

		chunks, err := s.Split(ctx, doc)
		if err != nil {
			t.Fatal(err)
		}
		for i, c := range chunks {
			if c.Position == nil {
				t.Fatalf("chunk %d: expected position", i)
			}
			got := text[c.Position.StartByte:c.Position.EndByte]
			if got != c.AsText() {
				t.Errorf("chunk %d: position slice %q does not match text %q", i, got, c.AsText())
			}
		}
	})

	t.Run("metadata is copied not shared", func(t *testing.T) {
		s := NewSentence(SentenceConfig{Size: 20})
		doc := mustDoc(t, "d1", "First. Second. Third.", dory.WithMetadata("key", "val"))

		chunks, err := s.Split(ctx, doc)
		if err != nil {
			t.Fatal(err)
		}
		if len(chunks) < 2 {
			t.Fatalf("need at least 2 chunks")
		}
		chunks[0].Metadata()["key"] = "changed"
		if chunks[1].Metadata()["key"] != "val" {
			t.Error("metadata is shared between chunks, should be independent copies")
		}
	})

	t.Run("source URI is propagated", func(t *testing.T) {
		s := NewSentence(SentenceConfig{Size: 100})
		doc := mustDoc(t, "d1", "Hello.", dory.WithSourceURI("s3://bucket/file.txt"))

		chunks, err := s.Split(ctx, doc)
		if err != nil {
			t.Fatal(err)
		}
		if chunks[0].SourceURI() != "s3://bucket/file.txt" {
			t.Errorf("got %q, want %q", chunks[0].SourceURI(), "s3://bucket/file.txt")
		}
	})

	t.Run("handles newline sentence endings", func(t *testing.T) {
		s := NewSentence(SentenceConfig{Size: 30})
		doc := mustDoc(t, "d1", "First sentence.\nSecond sentence.\nThird.")

		chunks, err := s.Split(ctx, doc)
		if err != nil {
			t.Fatal(err)
		}
		if len(chunks) < 2 {
			t.Fatalf("got %d chunks, want at least 2", len(chunks))
		}
	})
}
