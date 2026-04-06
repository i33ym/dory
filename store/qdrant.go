package store

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/i33ym/dory"
)

// QdrantConfig holds configuration for a Qdrant vector store.
type QdrantConfig struct {
	// URL is the base URL of the Qdrant server (e.g. "http://localhost:6333").
	URL string

	// CollectionName is the name of the Qdrant collection.
	CollectionName string

	// Dimensions is the size of the embedding vectors (e.g. 1536).
	Dimensions int

	// APIKey is an optional API key for authentication.
	APIKey string
}

// Qdrant is a VectorStore backed by the Qdrant vector database via its HTTP REST API.
type Qdrant struct {
	url        string
	collection string
	dimensions int
	apiKey     string
	client     *http.Client
}

// NewQdrant creates a new Qdrant-backed vector store.
func NewQdrant(config QdrantConfig) (*Qdrant, error) {
	if config.URL == "" {
		return nil, fmt.Errorf("dory/store: QdrantConfig.URL must not be empty")
	}
	if config.CollectionName == "" {
		return nil, fmt.Errorf("dory/store: QdrantConfig.CollectionName must not be empty")
	}
	if config.Dimensions <= 0 {
		return nil, fmt.Errorf("dory/store: QdrantConfig.Dimensions must be > 0")
	}
	return &Qdrant{
		url:        strings.TrimRight(config.URL, "/"),
		collection: config.CollectionName,
		dimensions: config.Dimensions,
		apiKey:     config.APIKey,
		client:     &http.Client{},
	}, nil
}

// EnsureCollection creates the Qdrant collection if it does not already exist.
func (q *Qdrant) EnsureCollection(ctx context.Context) error {
	body := map[string]any{
		"vectors": map[string]any{
			"size":     q.dimensions,
			"distance": "Cosine",
		},
	}
	_, err := q.doRequest(ctx, http.MethodPut, q.collectionURL(), body)
	return err
}

// Store upserts chunks as points in the Qdrant collection.
func (q *Qdrant) Store(ctx context.Context, chunks []*dory.Chunk) error {
	if len(chunks) == 0 {
		return nil
	}

	points := make([]qdrantPoint, len(chunks))
	for i, c := range chunks {
		payload := make(map[string]any)
		payload["source_doc_id"] = c.SourceDocumentID()
		payload["source_uri"] = c.SourceURI()
		payload["content"] = c.AsText()
		if meta := c.Metadata(); meta != nil {
			for k, v := range meta {
				payload[k] = v
			}
		}

		vec := make([]float64, len(c.Vector))
		for j, f := range c.Vector {
			vec[j] = float64(f)
		}

		points[i] = qdrantPoint{
			ID:      c.ID(),
			Vector:  vec,
			Payload: payload,
		}
	}

	body := map[string]any{
		"points": points,
	}
	_, err := q.doRequest(ctx, http.MethodPut, q.collectionURL()+"/points", body)
	return err
}

// Search finds the top-k chunks by cosine similarity in the Qdrant collection.
func (q *Qdrant) Search(ctx context.Context, req dory.SearchRequest) ([]dory.ScoredChunk, error) {
	topK := req.TopK
	if topK <= 0 {
		topK = 10
	}

	vec := make([]float64, len(req.QueryVector))
	for i, f := range req.QueryVector {
		vec[i] = float64(f)
	}

	body := map[string]any{
		"nearest": map[string]any{
			"vector": vec,
		},
		"limit":        topK,
		"with_payload": true,
	}

	if req.Filter != nil {
		body["filter"] = buildQdrantFilter(req.Filter)
	}

	respBody, err := q.doRequest(ctx, http.MethodPost, q.collectionURL()+"/points/query", body)
	if err != nil {
		return nil, err
	}

	var resp qdrantQueryResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("dory/store: unmarshal search response: %w", err)
	}

	results := make([]dory.ScoredChunk, 0, len(resp.Result.Points))
	for _, pt := range resp.Result.Points {
		sourceDocID, _ := pt.Payload["source_doc_id"].(string)
		content, _ := pt.Payload["content"].(string)

		// Build metadata from payload, excluding internal fields.
		meta := make(map[string]any)
		for k, v := range pt.Payload {
			if k == "source_doc_id" || k == "source_uri" || k == "content" {
				continue
			}
			meta[k] = v
		}

		chunk := dory.NewChunk(pt.ID, sourceDocID, content, meta)
		results = append(results, dory.ScoredChunk{Chunk: chunk, Score: pt.Score})
	}

	return results, nil
}

// Delete removes points by their IDs from the Qdrant collection.
func (q *Qdrant) Delete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	body := map[string]any{
		"points": ids,
	}
	_, err := q.doRequest(ctx, http.MethodPost, q.collectionURL()+"/points/delete", body)
	return err
}

func (q *Qdrant) collectionURL() string {
	return fmt.Sprintf("%s/collections/%s", q.url, q.collection)
}

func (q *Qdrant) doRequest(ctx context.Context, method, url string, body any) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("dory/store: marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("dory/store: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if q.apiKey != "" {
		req.Header.Set("api-key", q.apiKey)
	}

	resp, err := q.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("dory/store: request %s %s: %w", method, url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("dory/store: read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("dory/store: qdrant %s %s returned %d: %s", method, url, resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// buildQdrantFilter translates a dory.MetadataFilter into Qdrant's filter format.
func buildQdrantFilter(f *dory.MetadataFilter) map[string]any {
	switch f.Op {
	case dory.FilterOpEq:
		return map[string]any{
			"must": []any{
				map[string]any{
					"key":   f.Field,
					"match": map[string]any{"value": f.Value},
				},
			},
		}
	case dory.FilterOpIn:
		vals, _ := f.Value.([]string)
		anyVals := make([]any, len(vals))
		for i, v := range vals {
			anyVals[i] = v
		}
		return map[string]any{
			"must": []any{
				map[string]any{
					"key":   f.Field,
					"match": map[string]any{"any": anyVals},
				},
			},
		}
	case dory.FilterOpAnyOf:
		vals, _ := f.Value.([]string)
		anyVals := make([]any, len(vals))
		for i, v := range vals {
			anyVals[i] = v
		}
		return map[string]any{
			"must": []any{
				map[string]any{
					"key":   f.Field,
					"match": map[string]any{"any": anyVals},
				},
			},
		}
	default:
		return nil
	}
}

// Qdrant API types

type qdrantPoint struct {
	ID      string         `json:"id"`
	Vector  []float64      `json:"vector"`
	Payload map[string]any `json:"payload"`
}

type qdrantQueryResponse struct {
	Result qdrantQueryResult `json:"result"`
}

type qdrantQueryResult struct {
	Points []qdrantScoredPoint `json:"points"`
}

type qdrantScoredPoint struct {
	ID      string         `json:"id"`
	Score   float64        `json:"score"`
	Payload map[string]any `json:"payload"`
}
