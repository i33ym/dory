package embed

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOllama_Embed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/embed" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}

		var req ollamaRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if req.Model != "nomic-embed-text" {
			t.Errorf("got model %q, want %q", req.Model, "nomic-embed-text")
		}

		resp := ollamaResponse{
			Embeddings: [][]float64{{0.1, 0.2, 0.3}},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	e := NewOllama(OllamaConfig{
		Model:      "nomic-embed-text",
		URL:        server.URL,
		Dimensions: 3,
	})

	vec, err := e.Embed(context.Background(), "hello world")
	if err != nil {
		t.Fatal(err)
	}
	if len(vec) != 3 {
		t.Fatalf("got %d dimensions, want 3", len(vec))
	}
	if vec[0] != float32(0.1) || vec[1] != float32(0.2) || vec[2] != float32(0.3) {
		t.Errorf("unexpected vector values: %v", vec)
	}
}

func TestOllama_EmbedBatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req ollamaRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}

		// Input should be an array of strings for batch.
		inputs, ok := req.Input.([]any)
		if !ok {
			t.Fatalf("expected array input, got %T", req.Input)
		}
		if len(inputs) != 2 {
			t.Fatalf("got %d inputs, want 2", len(inputs))
		}

		resp := ollamaResponse{
			Embeddings: [][]float64{
				{0.1, 0.2, 0.3, 0.4},
				{0.5, 0.6, 0.7, 0.8},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	e := NewOllama(OllamaConfig{
		Model:      "nomic-embed-text",
		URL:        server.URL,
		Dimensions: 4,
	})

	vecs, err := e.EmbedBatch(context.Background(), []string{"hello", "world"})
	if err != nil {
		t.Fatal(err)
	}
	if len(vecs) != 2 {
		t.Fatalf("got %d vectors, want 2", len(vecs))
	}
	for i, vec := range vecs {
		if len(vec) != 4 {
			t.Errorf("vector %d: got %d dimensions, want 4", i, len(vec))
		}
	}

	// Verify values.
	if vecs[0][0] != float32(0.1) {
		t.Errorf("vecs[0][0] = %f, want 0.1", vecs[0][0])
	}
	if vecs[1][0] != float32(0.5) {
		t.Errorf("vecs[1][0] = %f, want 0.5", vecs[1][0])
	}
}

func TestOllama_EmbedBatch_Empty(t *testing.T) {
	e := NewOllama(OllamaConfig{
		Model:      "nomic-embed-text",
		Dimensions: 4,
	})

	vecs, err := e.EmbedBatch(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if vecs != nil {
		t.Errorf("expected nil for empty input, got %v", vecs)
	}
}

func TestOllama_Dimensions(t *testing.T) {
	e := NewOllama(OllamaConfig{
		Model:      "nomic-embed-text",
		Dimensions: 768,
	})
	if e.Dimensions() != 768 {
		t.Errorf("got %d, want 768", e.Dimensions())
	}
}

func TestOllama_DefaultURL(t *testing.T) {
	e := NewOllama(OllamaConfig{
		Model:      "nomic-embed-text",
		Dimensions: 768,
	})
	if e.url != "http://localhost:11434" {
		t.Errorf("got %q, want %q", e.url, "http://localhost:11434")
	}
}

func TestOllama_Embed_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	e := NewOllama(OllamaConfig{
		Model:      "nomic-embed-text",
		URL:        server.URL,
		Dimensions: 3,
	})

	_, err := e.Embed(context.Background(), "hello")
	if err == nil {
		t.Fatal("expected error for server error response")
	}
}
