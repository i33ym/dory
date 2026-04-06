package retrieve

import (
	"context"
	"errors"
	"testing"

	"github.com/i33ym/dory"
)

func TestStructured_Retrieve(t *testing.T) {
	ctx := context.Background()

	s := NewStructured(StructuredConfig{
		TextToSQL: func(_ context.Context, question string) (string, error) {
			return "SELECT name, age FROM users WHERE age > 30", nil
		},
		ExecSQL: func(_ context.Context, sql string) ([]map[string]any, error) {
			return []map[string]any{
				{"name": "Alice", "age": 35},
				{"name": "Bob", "age": 42},
			}, nil
		},
		SourceDocID: "users-db",
	})

	results, err := s.Retrieve(ctx, dory.Query{Text: "who is older than 30?", TopK: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}

	// Check that results are StructuredRow units.
	for _, r := range results {
		if r.SourceDocumentID() != "users-db" {
			t.Errorf("source doc = %s, want users-db", r.SourceDocumentID())
		}
		if r.Score() != 1.0 {
			t.Errorf("score = %f, want 1.0", r.Score())
		}
	}
}

func TestStructured_TextToSQLError(t *testing.T) {
	ctx := context.Background()

	s := NewStructured(StructuredConfig{
		TextToSQL: func(_ context.Context, _ string) (string, error) {
			return "", errors.New("cannot parse question")
		},
		ExecSQL:     func(_ context.Context, _ string) ([]map[string]any, error) { return nil, nil },
		SourceDocID: "db",
	})

	_, err := s.Retrieve(ctx, dory.Query{Text: "gibberish", TopK: 5})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestStructured_ExecSQLError(t *testing.T) {
	ctx := context.Background()

	s := NewStructured(StructuredConfig{
		TextToSQL: func(_ context.Context, _ string) (string, error) {
			return "SELECT 1", nil
		},
		ExecSQL: func(_ context.Context, _ string) ([]map[string]any, error) {
			return nil, errors.New("connection refused")
		},
		SourceDocID: "db",
	})

	_, err := s.Retrieve(ctx, dory.Query{Text: "test", TopK: 5})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
