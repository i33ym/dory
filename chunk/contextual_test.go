package chunk

import (
	"context"
	"testing"
)

func TestContextual_Split(t *testing.T) {
	ctx := context.Background()

	t.Run("ContextPrefix is set on chunks", func(t *testing.T) {
		contextFn := func(_ context.Context, docContent, chunkText string) (string, error) {
			return "This chunk discusses: " + chunkText[:3], nil
		}

		s := NewContextual(ContextualConfig{
			Size:        5,
			ContextFunc: contextFn,
		})
		doc := mustDoc(t, "d1", "abcdefghij")

		chunks, err := s.Split(ctx, doc)
		if err != nil {
			t.Fatal(err)
		}

		if len(chunks) != 2 {
			t.Fatalf("got %d chunks, want 2", len(chunks))
		}

		if chunks[0].ContextPrefix != "This chunk discusses: abc" {
			t.Errorf("chunk 0 ContextPrefix: got %q", chunks[0].ContextPrefix)
		}
		if chunks[1].ContextPrefix != "This chunk discusses: fgh" {
			t.Errorf("chunk 1 ContextPrefix: got %q", chunks[1].ContextPrefix)
		}
	})

	t.Run("AsText includes context prefix", func(t *testing.T) {
		contextFn := func(_ context.Context, _, _ string) (string, error) {
			return "Context:", nil
		}

		s := NewContextual(ContextualConfig{
			Size:        100,
			ContextFunc: contextFn,
		})
		doc := mustDoc(t, "d1", "hello world")

		chunks, err := s.Split(ctx, doc)
		if err != nil {
			t.Fatal(err)
		}

		if len(chunks) != 1 {
			t.Fatalf("got %d chunks, want 1", len(chunks))
		}

		want := "Context: hello world"
		if chunks[0].AsText() != want {
			t.Errorf("AsText: got %q, want %q", chunks[0].AsText(), want)
		}
	})

	t.Run("empty content returns nil", func(t *testing.T) {
		contextFn := func(_ context.Context, _, _ string) (string, error) {
			return "ctx", nil
		}
		s := NewContextual(ContextualConfig{ContextFunc: contextFn})
		doc := mustDoc(t, "d1", "")

		chunks, err := s.Split(ctx, doc)
		if err != nil {
			t.Fatal(err)
		}
		if chunks != nil {
			t.Fatalf("got %d chunks, want nil", len(chunks))
		}
	})

	t.Run("position is set correctly", func(t *testing.T) {
		contextFn := func(_ context.Context, _, _ string) (string, error) {
			return "pfx", nil
		}

		s := NewContextual(ContextualConfig{
			Size:        5,
			ContextFunc: contextFn,
		})
		doc := mustDoc(t, "d1", "abcdefghij")

		chunks, err := s.Split(ctx, doc)
		if err != nil {
			t.Fatal(err)
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
}
