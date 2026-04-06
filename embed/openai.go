// Package embed provides embedder implementations for Dory.
// Each implementation satisfies the [dory.Embedder] interface.
package embed

import (
	"context"
	"fmt"

	"github.com/openai/openai-go"
)

// OpenAI implements the [dory.Embedder] interface using the official OpenAI Go SDK.
// Reads the API key from the OPENAI_API_KEY environment variable by default.
type OpenAI struct {
	model      openai.EmbeddingModel
	dimensions int
	client     openai.Client
}

// NewOpenAI creates a new OpenAI embedder for the given model name.
func NewOpenAI(model string) *OpenAI {
	dims := 1536
	switch model {
	case "text-embedding-3-small":
		dims = 1536
	case "text-embedding-3-large":
		dims = 3072
	case "text-embedding-ada-002":
		dims = 1536
	}
	return &OpenAI{
		model:      openai.EmbeddingModel(model),
		dimensions: dims,
		client:     openai.NewClient(),
	}
}

// Embed returns the vector representation of the given text.
func (o *OpenAI) Embed(ctx context.Context, text string) ([]float32, error) {
	resp, err := o.client.Embeddings.New(ctx, openai.EmbeddingNewParams{
		Input: openai.EmbeddingNewParamsInputUnion{
			OfString: openai.String(text),
		},
		Model: o.model,
	})
	if err != nil {
		return nil, fmt.Errorf("embed: %w", err)
	}
	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("embed: no embedding returned")
	}
	return toFloat32(resp.Data[0].Embedding), nil
}

// EmbedBatch embeds multiple texts in a single API call.
func (o *OpenAI) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	resp, err := o.client.Embeddings.New(ctx, openai.EmbeddingNewParams{
		Input: openai.EmbeddingNewParamsInputUnion{
			OfArrayOfStrings: texts,
		},
		Model: o.model,
	})
	if err != nil {
		return nil, fmt.Errorf("embed: %w", err)
	}

	results := make([][]float32, len(texts))
	for _, d := range resp.Data {
		if d.Index < int64(len(results)) {
			results[d.Index] = toFloat32(d.Embedding)
		}
	}
	return results, nil
}

// Dimensions returns the dimensionality of the vectors this embedder produces.
func (o *OpenAI) Dimensions() int {
	return o.dimensions
}

func toFloat32(f64 []float64) []float32 {
	f32 := make([]float32, len(f64))
	for i, v := range f64 {
		f32[i] = float32(v)
	}
	return f32
}
