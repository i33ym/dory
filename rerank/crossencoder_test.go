package rerank

import (
	"context"
	"fmt"
	"testing"

	dory "github.com/i33ym/dory"
)

func TestCrossEncoder_Ordering(t *testing.T) {
	scoreFunc := func(_ context.Context, query string, doc string) (float64, error) {
		scores := map[string]float64{
			"alpha": 0.3,
			"beta":  0.9,
			"gamma": 0.6,
		}
		return scores[doc], nil
	}

	ce := NewCrossEncoder(CrossEncoderConfig{ScoreFunc: scoreFunc})

	units := []dory.RetrievedUnit{
		dory.NewChunk("1", "doc1", "alpha", nil).WithScore("test", 0.5),
		dory.NewChunk("2", "doc1", "beta", nil).WithScore("test", 0.5),
		dory.NewChunk("3", "doc1", "gamma", nil).WithScore("test", 0.5),
	}

	result, err := ce.Rerank(context.Background(), "query", units)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 3 {
		t.Fatalf("expected 3 results, got %d", len(result))
	}

	// Should be sorted: beta (0.9), gamma (0.6), alpha (0.3)
	expected := []string{"2", "3", "1"}
	for i, id := range expected {
		if result[i].ID() != id {
			t.Errorf("position %d: expected ID %s, got %s", i, id, result[i].ID())
		}
	}
}

func TestCrossEncoder_TopK(t *testing.T) {
	scoreFunc := func(_ context.Context, _ string, doc string) (float64, error) {
		scores := map[string]float64{
			"a": 0.9,
			"b": 0.7,
			"c": 0.5,
			"d": 0.3,
		}
		return scores[doc], nil
	}

	ce := NewCrossEncoder(CrossEncoderConfig{ScoreFunc: scoreFunc, TopK: 2})

	units := make([]dory.RetrievedUnit, 4)
	for i, text := range []string{"a", "b", "c", "d"} {
		units[i] = dory.NewChunk(fmt.Sprintf("%d", i), "doc1", text, nil).WithScore("test", 0.5)
	}

	result, err := ce.Rerank(context.Background(), "query", units)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result))
	}

	if result[0].ID() != "0" || result[1].ID() != "1" {
		t.Errorf("expected IDs [0, 1], got [%s, %s]", result[0].ID(), result[1].ID())
	}
}

func TestCrossEncoder_Threshold(t *testing.T) {
	scoreFunc := func(_ context.Context, _ string, doc string) (float64, error) {
		scores := map[string]float64{
			"high":   0.8,
			"medium": 0.5,
			"low":    0.2,
		}
		return scores[doc], nil
	}

	ce := NewCrossEncoder(CrossEncoderConfig{ScoreFunc: scoreFunc, Threshold: 0.4})

	units := []dory.RetrievedUnit{
		dory.NewChunk("1", "doc1", "high", nil).WithScore("test", 0.5),
		dory.NewChunk("2", "doc1", "medium", nil).WithScore("test", 0.5),
		dory.NewChunk("3", "doc1", "low", nil).WithScore("test", 0.5),
	}

	result, err := ce.Rerank(context.Background(), "query", units)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result))
	}

	if result[0].ID() != "1" || result[1].ID() != "2" {
		t.Errorf("expected IDs [1, 2], got [%s, %s]", result[0].ID(), result[1].ID())
	}
}

func TestCrossEncoder_ScoreProvenance(t *testing.T) {
	scoreFunc := func(_ context.Context, _ string, _ string) (float64, error) {
		return 0.75, nil
	}

	ce := NewCrossEncoder(CrossEncoderConfig{ScoreFunc: scoreFunc})

	units := []dory.RetrievedUnit{
		dory.NewChunk("1", "doc1", "text", nil).WithScore("test", 0.5),
	}

	result, err := ce.Rerank(context.Background(), "query", units)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	scores := result[0].Scores()
	if len(scores) != 2 {
		t.Fatalf("expected 2 score entries, got %d", len(scores))
	}

	if scores[0].Stage != "test" || scores[0].Score != 0.5 {
		t.Errorf("first score: expected test/0.5, got %s/%f", scores[0].Stage, scores[0].Score)
	}
	if scores[1].Stage != "crossencoder" || scores[1].Score != 0.75 {
		t.Errorf("second score: expected crossencoder/0.75, got %s/%f", scores[1].Stage, scores[1].Score)
	}
}

func TestCrossEncoder_Empty(t *testing.T) {
	ce := NewCrossEncoder(CrossEncoderConfig{
		ScoreFunc: func(_ context.Context, _ string, _ string) (float64, error) {
			return 0, nil
		},
	})

	result, err := ce.Rerank(context.Background(), "query", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestCrossEncoder_ErrorPropagation(t *testing.T) {
	scoreFunc := func(_ context.Context, _ string, _ string) (float64, error) {
		return 0, fmt.Errorf("model unavailable")
	}

	ce := NewCrossEncoder(CrossEncoderConfig{ScoreFunc: scoreFunc})
	units := []dory.RetrievedUnit{
		dory.NewChunk("1", "doc1", "text", nil).WithScore("test", 0.5),
	}

	_, err := ce.Rerank(context.Background(), "query", units)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
