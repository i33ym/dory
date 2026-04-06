package embed

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// OllamaConfig holds the configuration for an Ollama embedder.
type OllamaConfig struct {
	// Model is the Ollama model name, e.g. "nomic-embed-text".
	Model string

	// URL is the base URL of the Ollama HTTP API.
	// Defaults to "http://localhost:11434" if empty.
	URL string

	// Dimensions is the dimensionality of the vectors this model produces.
	Dimensions int
}

// Ollama implements the [dory.Embedder] interface using the Ollama local HTTP API.
type Ollama struct {
	model      string
	url        string
	dimensions int
	client     *http.Client
}

// NewOllama creates a new Ollama embedder with the given configuration.
func NewOllama(config OllamaConfig) *Ollama {
	url := config.URL
	if url == "" {
		url = "http://localhost:11434"
	}
	return &Ollama{
		model:      config.Model,
		url:        url,
		dimensions: config.Dimensions,
		client:     &http.Client{},
	}
}

// ollamaRequest is the JSON body sent to the Ollama embed API.
type ollamaRequest struct {
	Model string `json:"model"`
	Input any    `json:"input"`
}

// ollamaResponse is the JSON response from the Ollama embed API.
type ollamaResponse struct {
	Embeddings [][]float64 `json:"embeddings"`
}

// Embed returns the vector representation of the given text.
func (o *Ollama) Embed(ctx context.Context, text string) ([]float32, error) {
	vecs, err := o.embed(ctx, text)
	if err != nil {
		return nil, err
	}
	if len(vecs) == 0 {
		return nil, fmt.Errorf("embed: no embedding returned")
	}
	return toFloat32(vecs[0]), nil
}

// EmbedBatch embeds multiple texts in a single API call.
func (o *Ollama) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	vecs, err := o.embed(ctx, texts)
	if err != nil {
		return nil, err
	}
	results := make([][]float32, len(vecs))
	for i, v := range vecs {
		results[i] = toFloat32(v)
	}
	return results, nil
}

// Dimensions returns the dimensionality of the vectors this embedder produces.
func (o *Ollama) Dimensions() int {
	return o.dimensions
}

func (o *Ollama) embed(ctx context.Context, input any) ([][]float64, error) {
	body, err := json.Marshal(ollamaRequest{
		Model: o.model,
		Input: input,
	})
	if err != nil {
		return nil, fmt.Errorf("embed: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.url+"/api/embed", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("embed: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embed: ollama returned status %d", resp.StatusCode)
	}

	var result ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("embed: decode response: %w", err)
	}

	return result.Embeddings, nil
}
