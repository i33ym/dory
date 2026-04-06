// Package eval provides the evaluation pipeline for Dory.
// It measures retrieval quality with quantitative metrics inspired by RAGAS:
// context precision, context recall, faithfulness, and answer relevance.
package eval

import (
	"context"
	"errors"
	"strings"

	dory "github.com/i33ym/dory"
)

// RetrieverEvaluatorConfig configures a RetrieverEvaluator.
type RetrieverEvaluatorConfig struct {
	// Retriever is the retrieval pipeline to evaluate. Required.
	Retriever dory.Retriever

	// TopK is the maximum number of results to request from the retriever.
	// Defaults to 10 if zero.
	TopK int

	// Generator, if set, produces an answer from a question and concatenated
	// context. This enables faithfulness and answer relevance metrics.
	// If nil, those metrics are skipped.
	Generator func(ctx context.Context, question string, context string) (string, error)
}

// RetrieverEvaluator implements dory.Evaluator by running a retriever against
// test cases and computing context precision and context recall. When a
// Generator is provided it also generates answers (faithfulness and answer
// relevance are left as future work requiring LLM-as-judge scoring).
type RetrieverEvaluator struct {
	retriever dory.Retriever
	topK      int
	generator func(ctx context.Context, question string, context string) (string, error)
}

// NewRetrieverEvaluator creates a RetrieverEvaluator from the given config.
// It returns an error if config.Retriever is nil.
func NewRetrieverEvaluator(config RetrieverEvaluatorConfig) (*RetrieverEvaluator, error) {
	if config.Retriever == nil {
		return nil, errors.New("eval: Retriever is required")
	}
	topK := config.TopK
	if topK <= 0 {
		topK = 10
	}
	return &RetrieverEvaluator{
		retriever: config.Retriever,
		topK:      topK,
		generator: config.Generator,
	}, nil
}

// Evaluate runs each test case through the retriever, optionally generates an
// answer, and computes retrieval metrics.
func (e *RetrieverEvaluator) Evaluate(ctx context.Context, cases []dory.TestCase) ([]dory.EvalResult, error) {
	results := make([]dory.EvalResult, 0, len(cases))

	for _, tc := range cases {
		units, err := e.retriever.Retrieve(ctx, dory.Query{
			Text: tc.Question,
			TopK: e.topK,
		})
		if err != nil {
			return nil, err
		}

		result := dory.EvalResult{
			TestCase:       tc,
			RetrievedUnits: units,
		}

		// Compute context precision and recall when relevant doc IDs are provided.
		if len(tc.RelevantDocumentIDs) > 0 {
			cp := contextPrecision(units, tc.RelevantDocumentIDs)
			cr := contextRecall(units, tc.RelevantDocumentIDs)
			result.Metrics.ContextPrecision = &cp
			result.Metrics.ContextRecall = &cr
		}

		// Generate an answer if a generator is available.
		if e.generator != nil {
			ctxText := buildContext(units)
			answer, err := e.generator(ctx, tc.Question, ctxText)
			if err != nil {
				return nil, err
			}
			result.GeneratedAnswer = answer

			// Faithfulness and AnswerRelevance require LLM-as-judge scoring.
			// Left as nil (future work).
		}

		results = append(results, result)
	}

	return results, nil
}

// contextPrecision returns the fraction of retrieved units whose
// SourceDocumentID appears in relevantIDs. Returns 0 when units is empty.
func contextPrecision(units []dory.RetrievedUnit, relevantIDs []string) float64 {
	if len(units) == 0 {
		return 0
	}
	relevant := make(map[string]struct{}, len(relevantIDs))
	for _, id := range relevantIDs {
		relevant[id] = struct{}{}
	}
	hits := 0
	for _, u := range units {
		if _, ok := relevant[u.SourceDocumentID()]; ok {
			hits++
		}
	}
	return float64(hits) / float64(len(units))
}

// contextRecall returns the fraction of relevantIDs that appear among the
// retrieved units' SourceDocumentIDs. Returns 0 when relevantIDs is empty.
func contextRecall(units []dory.RetrievedUnit, relevantIDs []string) float64 {
	if len(relevantIDs) == 0 {
		return 0
	}
	found := make(map[string]struct{}, len(units))
	for _, u := range units {
		found[u.SourceDocumentID()] = struct{}{}
	}
	hits := 0
	for _, id := range relevantIDs {
		if _, ok := found[id]; ok {
			hits++
		}
	}
	return float64(hits) / float64(len(relevantIDs))
}

// buildContext concatenates the text of all retrieved units into a single
// context string separated by double newlines.
func buildContext(units []dory.RetrievedUnit) string {
	parts := make([]string, len(units))
	for i, u := range units {
		parts[i] = u.AsText()
	}
	return strings.Join(parts, "\n\n")
}
