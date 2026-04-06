package chunk

import (
	"context"
	"testing"
)

func TestProposition_Split(t *testing.T) {
	ctx := context.Background()

	t.Run("each proposition becomes a chunk", func(t *testing.T) {
		extractFn := func(_ context.Context, _ string) ([]string, error) {
			return []string{
				"The sky is blue.",
				"Water is wet.",
				"Fire is hot.",
			}, nil
		}

		s := NewProposition(PropositionConfig{ExtractFunc: extractFn})
		doc := mustDoc(t, "d1", "The sky is blue and water is wet. Fire is hot.")

		chunks, err := s.Split(ctx, doc)
		if err != nil {
			t.Fatal(err)
		}

		if len(chunks) != 3 {
			t.Fatalf("got %d chunks, want 3", len(chunks))
		}

		want := []string{"The sky is blue.", "Water is wet.", "Fire is hot."}
		for i, c := range chunks {
			if c.AsText() != want[i] {
				t.Errorf("chunk %d: got %q, want %q", i, c.AsText(), want[i])
			}
		}
	})

	t.Run("position covers entire document", func(t *testing.T) {
		content := "Some document content here."
		extractFn := func(_ context.Context, _ string) ([]string, error) {
			return []string{"Prop 1.", "Prop 2."}, nil
		}

		s := NewProposition(PropositionConfig{ExtractFunc: extractFn})
		doc := mustDoc(t, "d1", content)

		chunks, err := s.Split(ctx, doc)
		if err != nil {
			t.Fatal(err)
		}

		for i, c := range chunks {
			if c.Position.StartByte != 0 {
				t.Errorf("chunk %d: StartByte = %d, want 0", i, c.Position.StartByte)
			}
			if c.Position.EndByte != len(content) {
				t.Errorf("chunk %d: EndByte = %d, want %d", i, c.Position.EndByte, len(content))
			}
		}
	})

	t.Run("chunk IDs are sequential", func(t *testing.T) {
		extractFn := func(_ context.Context, _ string) ([]string, error) {
			return []string{"A.", "B.", "C."}, nil
		}

		s := NewProposition(PropositionConfig{ExtractFunc: extractFn})
		doc := mustDoc(t, "doc", "text")

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
		extractFn := func(_ context.Context, _ string) ([]string, error) {
			return []string{"Prop."}, nil
		}

		s := NewProposition(PropositionConfig{ExtractFunc: extractFn})
		doc := mustDoc(t, "my-doc", "text")

		chunks, err := s.Split(ctx, doc)
		if err != nil {
			t.Fatal(err)
		}

		if chunks[0].SourceDocumentID() != "my-doc" {
			t.Errorf("got %q, want %q", chunks[0].SourceDocumentID(), "my-doc")
		}
	})

	t.Run("empty content returns nil", func(t *testing.T) {
		extractFn := func(_ context.Context, _ string) ([]string, error) {
			return []string{"Should not be called."}, nil
		}

		s := NewProposition(PropositionConfig{ExtractFunc: extractFn})
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
