package eval

import (
	"context"
	"math"
	"testing"

	dory "github.com/i33ym/dory"
)

// fakeRetriever returns a fixed set of chunks for any query.
type fakeRetriever struct {
	units []dory.RetrievedUnit
}

func (f *fakeRetriever) Retrieve(_ context.Context, _ dory.Query) ([]dory.RetrievedUnit, error) {
	return f.units, nil
}

func makeChunk(id, sourceDocID, text string) *dory.Chunk {
	return dory.NewChunk(id, sourceDocID, text, nil)
}

func floatClose(t *testing.T, name string, got, want, epsilon float64) {
	t.Helper()
	if math.Abs(got-want) > epsilon {
		t.Errorf("%s: got %f, want %f (epsilon %f)", name, got, want, epsilon)
	}
}

func TestContextPrecision(t *testing.T) {
	// 2 of 4 retrieved units are relevant.
	units := []dory.RetrievedUnit{
		makeChunk("c1", "doc-a", "chunk 1"),
		makeChunk("c2", "doc-b", "chunk 2"),
		makeChunk("c3", "doc-c", "chunk 3"),
		makeChunk("c4", "doc-d", "chunk 4"),
	}
	relevantIDs := []string{"doc-a", "doc-c"}

	got := contextPrecision(units, relevantIDs)
	floatClose(t, "contextPrecision", got, 0.5, 0.001)
}

func TestContextRecall(t *testing.T) {
	// 2 of 3 relevant docs appear in retrieved units.
	units := []dory.RetrievedUnit{
		makeChunk("c1", "doc-a", "chunk 1"),
		makeChunk("c2", "doc-b", "chunk 2"),
	}
	relevantIDs := []string{"doc-a", "doc-b", "doc-c"}

	got := contextRecall(units, relevantIDs)
	floatClose(t, "contextRecall", got, 2.0/3.0, 0.001)
}

func TestContextPrecisionEmpty(t *testing.T) {
	got := contextPrecision(nil, []string{"doc-a"})
	floatClose(t, "contextPrecision(empty units)", got, 0, 0.001)
}

func TestContextRecallEmpty(t *testing.T) {
	got := contextRecall([]dory.RetrievedUnit{makeChunk("c1", "doc-a", "x")}, nil)
	floatClose(t, "contextRecall(empty relevantIDs)", got, 0, 0.001)
}

func TestEvaluate_NoRelevantIDs(t *testing.T) {
	ret := &fakeRetriever{
		units: []dory.RetrievedUnit{
			makeChunk("c1", "doc-a", "chunk 1"),
		},
	}
	ev, err := NewRetrieverEvaluator(RetrieverEvaluatorConfig{Retriever: ret})
	if err != nil {
		t.Fatal(err)
	}

	results, err := ev.Evaluate(context.Background(), []dory.TestCase{
		{ID: "t1", Question: "What?"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Metrics.ContextPrecision != nil {
		t.Error("expected ContextPrecision to be nil when no relevant IDs")
	}
	if results[0].Metrics.ContextRecall != nil {
		t.Error("expected ContextRecall to be nil when no relevant IDs")
	}
}

func TestEvaluate_WithMetrics(t *testing.T) {
	ret := &fakeRetriever{
		units: []dory.RetrievedUnit{
			makeChunk("c1", "doc-a", "chunk 1"),
			makeChunk("c2", "doc-b", "chunk 2"),
			makeChunk("c3", "doc-c", "chunk 3"),
			makeChunk("c4", "doc-d", "chunk 4"),
		},
	}
	ev, err := NewRetrieverEvaluator(RetrieverEvaluatorConfig{Retriever: ret, TopK: 5})
	if err != nil {
		t.Fatal(err)
	}

	results, err := ev.Evaluate(context.Background(), []dory.TestCase{
		{
			ID:                  "t1",
			Question:            "Tell me about A and C",
			RelevantDocumentIDs: []string{"doc-a", "doc-c"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if r.Metrics.ContextPrecision == nil {
		t.Fatal("expected ContextPrecision to be non-nil")
	}
	floatClose(t, "ContextPrecision", *r.Metrics.ContextPrecision, 0.5, 0.001)

	if r.Metrics.ContextRecall == nil {
		t.Fatal("expected ContextRecall to be non-nil")
	}
	floatClose(t, "ContextRecall", *r.Metrics.ContextRecall, 1.0, 0.001)

	if r.GeneratedAnswer != "" {
		t.Error("expected empty GeneratedAnswer without generator")
	}
	if r.Metrics.Faithfulness != nil {
		t.Error("expected Faithfulness to be nil without generator")
	}
	if r.Metrics.AnswerRelevance != nil {
		t.Error("expected AnswerRelevance to be nil without generator")
	}
}

func TestEvaluate_WithGenerator(t *testing.T) {
	ret := &fakeRetriever{
		units: []dory.RetrievedUnit{
			makeChunk("c1", "doc-a", "The sky is blue."),
			makeChunk("c2", "doc-b", "Water is wet."),
		},
	}

	generator := func(_ context.Context, question string, ctxText string) (string, error) {
		return "Generated answer for: " + question, nil
	}

	ev, err := NewRetrieverEvaluator(RetrieverEvaluatorConfig{
		Retriever: ret,
		Generator: generator,
	})
	if err != nil {
		t.Fatal(err)
	}

	results, err := ev.Evaluate(context.Background(), []dory.TestCase{
		{
			ID:                  "t1",
			Question:            "What color is the sky?",
			ReferenceAnswer:     "Blue",
			RelevantDocumentIDs: []string{"doc-a"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if r.GeneratedAnswer == "" {
		t.Error("expected non-empty GeneratedAnswer with generator")
	}
	if r.GeneratedAnswer != "Generated answer for: What color is the sky?" {
		t.Errorf("unexpected answer: %s", r.GeneratedAnswer)
	}
	// Without JudgeFunc, Faithfulness and AnswerRelevance remain nil.
	if r.Metrics.Faithfulness != nil {
		t.Error("expected Faithfulness to be nil without JudgeFunc")
	}
	if r.Metrics.AnswerRelevance != nil {
		t.Error("expected AnswerRelevance to be nil without JudgeFunc")
	}
}

func TestEvaluate_WithJudgeFunc(t *testing.T) {
	ret := &fakeRetriever{
		units: []dory.RetrievedUnit{
			makeChunk("c1", "doc-a", "The sky is blue."),
			makeChunk("c2", "doc-b", "Water is wet."),
		},
	}

	generator := func(_ context.Context, question string, ctxText string) (string, error) {
		return "The sky is blue.", nil
	}

	judgeFunc := func(_ context.Context, prompt string) (string, error) {
		return "0.85", nil
	}

	ev, err := NewRetrieverEvaluator(RetrieverEvaluatorConfig{
		Retriever: ret,
		Generator: generator,
		JudgeFunc: judgeFunc,
	})
	if err != nil {
		t.Fatal(err)
	}

	results, err := ev.Evaluate(context.Background(), []dory.TestCase{
		{
			ID:                  "t1",
			Question:            "What color is the sky?",
			ReferenceAnswer:     "Blue",
			RelevantDocumentIDs: []string{"doc-a"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]

	if r.Metrics.Faithfulness == nil {
		t.Fatal("expected Faithfulness to be non-nil with JudgeFunc")
	}
	floatClose(t, "Faithfulness", *r.Metrics.Faithfulness, 0.85, 0.001)

	if r.Metrics.AnswerRelevance == nil {
		t.Fatal("expected AnswerRelevance to be non-nil with JudgeFunc")
	}
	floatClose(t, "AnswerRelevance", *r.Metrics.AnswerRelevance, 0.85, 0.001)
}

func TestParseScore(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"0.85", 0.85},
		{"Score: 0.7", 0.7},
		{"The score is 0.95 out of 1.0", 0.95},
		{"no numbers here", 0},
		{"1", 1.0},
	}
	for _, tc := range tests {
		got := parseScore(tc.input)
		if math.Abs(got-tc.want) > 0.001 {
			t.Errorf("parseScore(%q) = %f, want %f", tc.input, got, tc.want)
		}
	}
}

func TestNewRetrieverEvaluator_Validation(t *testing.T) {
	_, err := NewRetrieverEvaluator(RetrieverEvaluatorConfig{})
	if err == nil {
		t.Fatal("expected error for nil Retriever")
	}
}

func TestNewRetrieverEvaluator_DefaultTopK(t *testing.T) {
	ev, err := NewRetrieverEvaluator(RetrieverEvaluatorConfig{
		Retriever: &fakeRetriever{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if ev.topK != 10 {
		t.Errorf("expected default topK=10, got %d", ev.topK)
	}
}
