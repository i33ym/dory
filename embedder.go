package dory

import "context"

// Embedder transforms text into a dense vector representation.
// The library is agnostic about which model or provider is used —
// any implementation of this interface is interchangeable.
type Embedder interface {
	// Embed returns the vector representation of the given text.
	Embed(ctx context.Context, text string) ([]float32, error)

	// EmbedBatch embeds multiple texts in a single call.
	// Implementations that do not support native batching should
	// loop over Embed internally. Callers should prefer EmbedBatch
	// during ingestion to reduce API round-trips and cost.
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)

	// Dimensions returns the dimensionality of the vectors this embedder
	// produces. The vector store needs this at collection creation time.
	Dimensions() int
}
