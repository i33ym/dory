package dory

import "context"

// TestCase is a single evaluation example.
type TestCase struct {
	// ID uniquely identifies this test case for result tracking.
	ID string

	// Question is the natural language query to evaluate.
	Question string

	// ReferenceAnswer is a high-quality answer to the question.
	// Used to score faithfulness and answer relevance.
	ReferenceAnswer string

	// RelevantDocumentIDs, if provided, are the document IDs that
	// should appear in the retrieved context.
	// Used to score context precision and context recall.
	RelevantDocumentIDs []string
}

// EvalMetrics holds the computed scores for a single test case.
// All scores are in the range [0.0, 1.0]. A nil pointer means
// the metric was not requested or could not be computed.
type EvalMetrics struct {
	// ContextPrecision measures what fraction of retrieved chunks
	// were actually relevant to the question.
	ContextPrecision *float64

	// ContextRecall measures what fraction of the information needed
	// to answer the question was present in the retrieved chunks.
	ContextRecall *float64

	// Faithfulness measures whether the generated answer is supported
	// by the retrieved context rather than the model's parametric knowledge.
	Faithfulness *float64

	// AnswerRelevance measures whether the generated answer actually
	// addresses what the question asked.
	AnswerRelevance *float64
}

// EvalResult captures the full output of evaluating one TestCase.
type EvalResult struct {
	TestCase        TestCase
	RetrievedUnits  []RetrievedUnit
	GeneratedAnswer string
	Metrics         EvalMetrics
}

// Evaluator runs a retrieval pipeline against a set of test cases
// and produces scored results for each.
type Evaluator interface {
	Evaluate(ctx context.Context, cases []TestCase) ([]EvalResult, error)
}
