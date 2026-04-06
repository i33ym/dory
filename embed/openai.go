// Package embed provides embedder implementations for Dory.
// Each implementation satisfies the [dory.Embedder] interface.
package embed

import (
	"context"
	"fmt"
)

// OpenAI implements the [dory.Embedder] interface using OpenAI's embedding API.
// Requires the OPENAI_API_KEY environment variable to be set.
type OpenAI struct {
	model      string
	dimensions int
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
	return &OpenAI{model: model, dimensions: dims}
}

// Embed returns the vector representation of the given text.
func (o *OpenAI) Embed(_ context.Context, _ string) ([]float32, error) {
	return nil, fmt.Errorf("embed: OpenAI embedder not yet implemented — set OPENAI_API_KEY and implement API call")
}

// EmbedBatch embeds multiple texts in a single call.
func (o *OpenAI) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	results := make([][]float32, len(texts))
	for i, t := range texts {
		vec, err := o.Embed(ctx, t)
		if err != nil {
			return nil, err
		}
		results[i] = vec
	}
	return results, nil
}

// Dimensions returns the dimensionality of the vectors this embedder produces.
func (o *OpenAI) Dimensions() int {
	return o.dimensions
}
