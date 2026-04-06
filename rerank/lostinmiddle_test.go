package rerank

import (
	"context"
	"fmt"
	"testing"

	dory "github.com/i33ym/dory"
)

func TestLostInTheMiddle_Reordering(t *testing.T) {
	litm := NewLostInTheMiddle()

	// Create 5 units with descending scores: 0.9, 0.8, 0.7, 0.6, 0.5
	units := make([]dory.RetrievedUnit, 5)
	for i := 0; i < 5; i++ {
		score := 0.9 - float64(i)*0.1
		units[i] = dory.NewChunk(fmt.Sprintf("%d", i), "doc1", fmt.Sprintf("text%d", i), nil).
			WithScore("test", score)
	}

	result, err := litm.Rerank(context.Background(), "query", units)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 5 {
		t.Fatalf("expected 5 results, got %d", len(result))
	}

	// Input sorted by score desc: IDs [0(0.9), 1(0.8), 2(0.7), 3(0.6), 4(0.5)]
	// Alternating front/back placement:
	//   i=0 (0.9) -> front[0]
	//   i=1 (0.8) -> back[4]
	//   i=2 (0.7) -> front[1]
	//   i=3 (0.6) -> back[3]
	//   i=4 (0.5) -> front[2]
	// Result: [0, 2, 4, 3, 1]
	expectedIDs := []string{"0", "2", "4", "3", "1"}
	for i, id := range expectedIDs {
		if result[i].ID() != id {
			t.Errorf("position %d: expected ID %s, got %s", i, id, result[i].ID())
		}
	}

	// Best items should be at edges (positions 0 and 4).
	edgeScores := []float64{result[0].Score(), result[len(result)-1].Score()}
	middleScore := result[2].Score()
	for _, es := range edgeScores {
		if es <= middleScore {
			t.Errorf("edge score %f should be greater than middle score %f", es, middleScore)
		}
	}
}

func TestLostInTheMiddle_PreservesScores(t *testing.T) {
	litm := NewLostInTheMiddle()

	units := []dory.RetrievedUnit{
		dory.NewChunk("1", "doc1", "a", nil).WithScore("test", 0.9),
		dory.NewChunk("2", "doc1", "b", nil).WithScore("test", 0.3),
	}

	result, err := litm.Rerank(context.Background(), "query", units)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, u := range result {
		scores := u.Scores()
		last := scores[len(scores)-1]
		if last.Stage != "litm_reorder" {
			t.Errorf("expected stage litm_reorder, got %s", last.Stage)
		}
		// The litm_reorder score should equal the original score.
		prev := scores[len(scores)-2]
		if last.Score != prev.Score {
			t.Errorf("litm_reorder score %f should equal original score %f", last.Score, prev.Score)
		}
	}
}

func TestLostInTheMiddle_Empty(t *testing.T) {
	litm := NewLostInTheMiddle()

	result, err := litm.Rerank(context.Background(), "query", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestLostInTheMiddle_SingleItem(t *testing.T) {
	litm := NewLostInTheMiddle()

	units := []dory.RetrievedUnit{
		dory.NewChunk("1", "doc1", "only", nil).WithScore("test", 0.5),
	}

	result, err := litm.Rerank(context.Background(), "query", units)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}
	if result[0].ID() != "1" {
		t.Errorf("expected ID 1, got %s", result[0].ID())
	}
}

func TestLostInTheMiddle_EvenCount(t *testing.T) {
	litm := NewLostInTheMiddle()

	// 4 units with scores: 1.0, 0.8, 0.6, 0.4
	units := make([]dory.RetrievedUnit, 4)
	for i := 0; i < 4; i++ {
		score := 1.0 - float64(i)*0.2
		units[i] = dory.NewChunk(fmt.Sprintf("%d", i), "doc1", fmt.Sprintf("t%d", i), nil).
			WithScore("test", score)
	}

	result, err := litm.Rerank(context.Background(), "query", units)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Sorted desc: [0(1.0), 1(0.8), 2(0.6), 3(0.4)]
	// i=0 -> front[0], i=1 -> back[3], i=2 -> front[1], i=3 -> back[2]
	// Result: [0, 2, 3, 1]
	expectedIDs := []string{"0", "2", "3", "1"}
	for i, id := range expectedIDs {
		if result[i].ID() != id {
			t.Errorf("position %d: expected ID %s, got %s", i, id, result[i].ID())
		}
	}
}
