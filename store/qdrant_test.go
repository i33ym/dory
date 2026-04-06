package store

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/i33ym/dory"
)

func TestNewQdrant_Validation(t *testing.T) {
	t.Run("empty URL", func(t *testing.T) {
		_, err := NewQdrant(QdrantConfig{CollectionName: "test", Dimensions: 128})
		if err == nil {
			t.Fatal("expected error for empty URL")
		}
	})

	t.Run("empty collection name", func(t *testing.T) {
		_, err := NewQdrant(QdrantConfig{URL: "http://localhost:6333", Dimensions: 128})
		if err == nil {
			t.Fatal("expected error for empty collection name")
		}
	})

	t.Run("zero dimensions", func(t *testing.T) {
		_, err := NewQdrant(QdrantConfig{URL: "http://localhost:6333", CollectionName: "test"})
		if err == nil {
			t.Fatal("expected error for zero dimensions")
		}
	})

	t.Run("valid config", func(t *testing.T) {
		q, err := NewQdrant(QdrantConfig{
			URL:            "http://localhost:6333",
			CollectionName: "test",
			Dimensions:     128,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if q == nil {
			t.Fatal("expected non-nil Qdrant")
		}
	})
}

func TestQdrant_EnsureCollection(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path

		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		gotBody = body

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"result": true, "status": "ok"}`))
	}))
	defer srv.Close()

	q, err := NewQdrant(QdrantConfig{
		URL:            srv.URL,
		CollectionName: "my_collection",
		Dimensions:     384,
	})
	if err != nil {
		t.Fatalf("NewQdrant: %v", err)
	}

	err = q.EnsureCollection(context.Background())
	if err != nil {
		t.Fatalf("EnsureCollection: %v", err)
	}

	if gotMethod != http.MethodPut {
		t.Errorf("expected PUT, got %s", gotMethod)
	}
	if gotPath != "/collections/my_collection" {
		t.Errorf("expected /collections/my_collection, got %s", gotPath)
	}

	vectors, ok := gotBody["vectors"].(map[string]any)
	if !ok {
		t.Fatal("expected vectors in request body")
	}
	if size, _ := vectors["size"].(float64); int(size) != 384 {
		t.Errorf("expected dimensions 384, got %v", vectors["size"])
	}
	if dist, _ := vectors["distance"].(string); dist != "Cosine" {
		t.Errorf("expected Cosine distance, got %s", dist)
	}
}

func TestQdrant_Store(t *testing.T) {
	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"result": {"status": "completed"}, "status": "ok"}`))
	}))
	defer srv.Close()

	q, _ := NewQdrant(QdrantConfig{URL: srv.URL, CollectionName: "test", Dimensions: 3})

	c1 := dory.NewChunk("c1", "doc1", "hello world", map[string]any{"tenant": "acme"})
	c1.Vector = []float32{0.1, 0.2, 0.3}

	c2 := dory.NewChunk("c2", "doc1", "goodbye world", nil)
	c2.Vector = []float32{0.4, 0.5, 0.6}

	err := q.Store(context.Background(), []*dory.Chunk{c1, c2})
	if err != nil {
		t.Fatalf("Store: %v", err)
	}

	points, ok := gotBody["points"].([]any)
	if !ok {
		t.Fatal("expected points array in request body")
	}
	if len(points) != 2 {
		t.Fatalf("expected 2 points, got %d", len(points))
	}

	p1, _ := points[0].(map[string]any)
	if p1["id"] != "c1" {
		t.Errorf("expected id c1, got %v", p1["id"])
	}
	payload, _ := p1["payload"].(map[string]any)
	if payload["content"] != "hello world" {
		t.Errorf("expected content 'hello world', got %v", payload["content"])
	}
	if payload["tenant"] != "acme" {
		t.Errorf("expected tenant 'acme', got %v", payload["tenant"])
	}
}

func TestQdrant_StoreEmpty(t *testing.T) {
	q, _ := NewQdrant(QdrantConfig{URL: "http://localhost:6333", CollectionName: "test", Dimensions: 3})
	err := q.Store(context.Background(), nil)
	if err != nil {
		t.Fatalf("Store(nil): %v", err)
	}
}

func TestQdrant_Search(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/collections/test/points/query" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)

		// Verify filter is present when expected
		resp := map[string]any{
			"result": map[string]any{
				"points": []any{
					map[string]any{
						"id":    "c1",
						"score": 0.95,
						"payload": map[string]any{
							"source_doc_id": "doc1",
							"source_uri":    "",
							"content":       "hello world",
							"tenant":        "acme",
						},
					},
					map[string]any{
						"id":    "c2",
						"score": 0.80,
						"payload": map[string]any{
							"source_doc_id": "doc1",
							"source_uri":    "",
							"content":       "goodbye world",
						},
					},
				},
			},
			"status": "ok",
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	q, _ := NewQdrant(QdrantConfig{URL: srv.URL, CollectionName: "test", Dimensions: 3})

	results, err := q.Search(context.Background(), dory.SearchRequest{
		QueryVector: []float32{0.1, 0.2, 0.3},
		TopK:        5,
	})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	if results[0].Chunk.ID() != "c1" {
		t.Errorf("expected first result id c1, got %s", results[0].Chunk.ID())
	}
	if results[0].Score != 0.95 {
		t.Errorf("expected score 0.95, got %f", results[0].Score)
	}
	if results[0].Chunk.SourceDocumentID() != "doc1" {
		t.Errorf("expected source_doc_id doc1, got %s", results[0].Chunk.SourceDocumentID())
	}

	// Metadata should contain tenant but not internal fields.
	meta := results[0].Chunk.Metadata()
	if meta["tenant"] != "acme" {
		t.Errorf("expected tenant acme in metadata, got %v", meta["tenant"])
	}
	if _, ok := meta["content"]; ok {
		t.Error("content should not be in metadata")
	}
}

func TestQdrant_SearchWithFilter(t *testing.T) {
	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusOK)
		resp := map[string]any{
			"result": map[string]any{"points": []any{}},
			"status": "ok",
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	q, _ := NewQdrant(QdrantConfig{URL: srv.URL, CollectionName: "test", Dimensions: 3})

	_, err := q.Search(context.Background(), dory.SearchRequest{
		QueryVector: []float32{0.1, 0.2, 0.3},
		TopK:        5,
		Filter: &dory.MetadataFilter{
			Field: "tenant",
			Op:    dory.FilterOpEq,
			Value: "acme",
		},
	})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	filter, ok := gotBody["filter"].(map[string]any)
	if !ok {
		t.Fatal("expected filter in request body")
	}
	must, ok := filter["must"].([]any)
	if !ok || len(must) == 0 {
		t.Fatal("expected must array in filter")
	}
	condition, _ := must[0].(map[string]any)
	if condition["key"] != "tenant" {
		t.Errorf("expected filter key 'tenant', got %v", condition["key"])
	}
}

func TestQdrant_Delete(t *testing.T) {
	var gotBody map[string]any
	var gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"result": true, "status": "ok"}`))
	}))
	defer srv.Close()

	q, _ := NewQdrant(QdrantConfig{URL: srv.URL, CollectionName: "test", Dimensions: 3})

	err := q.Delete(context.Background(), []string{"c1", "c2"})
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if gotPath != "/collections/test/points/delete" {
		t.Errorf("expected /collections/test/points/delete, got %s", gotPath)
	}

	points, ok := gotBody["points"].([]any)
	if !ok {
		t.Fatal("expected points in request body")
	}
	if len(points) != 2 {
		t.Fatalf("expected 2 point ids, got %d", len(points))
	}
}

func TestQdrant_DeleteEmpty(t *testing.T) {
	q, _ := NewQdrant(QdrantConfig{URL: "http://localhost:6333", CollectionName: "test", Dimensions: 3})
	err := q.Delete(context.Background(), nil)
	if err != nil {
		t.Fatalf("Delete(nil): %v", err)
	}
}

func TestQdrant_APIKey(t *testing.T) {
	var gotAPIKey string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAPIKey = r.Header.Get("api-key")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"result": true, "status": "ok"}`))
	}))
	defer srv.Close()

	q, _ := NewQdrant(QdrantConfig{
		URL:            srv.URL,
		CollectionName: "test",
		Dimensions:     3,
		APIKey:         "secret-key",
	})

	_ = q.EnsureCollection(context.Background())

	if gotAPIKey != "secret-key" {
		t.Errorf("expected api-key 'secret-key', got %q", gotAPIKey)
	}
}

func TestQdrant_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"status": "error", "result": null}`))
	}))
	defer srv.Close()

	q, _ := NewQdrant(QdrantConfig{URL: srv.URL, CollectionName: "test", Dimensions: 3})

	err := q.EnsureCollection(context.Background())
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestBuildQdrantFilter(t *testing.T) {
	t.Run("eq", func(t *testing.T) {
		f := buildQdrantFilter(&dory.MetadataFilter{
			Field: "tenant",
			Op:    dory.FilterOpEq,
			Value: "acme",
		})
		must, _ := f["must"].([]any)
		if len(must) != 1 {
			t.Fatalf("expected 1 must condition, got %d", len(must))
		}
		cond, _ := must[0].(map[string]any)
		if cond["key"] != "tenant" {
			t.Errorf("expected key 'tenant', got %v", cond["key"])
		}
		match, _ := cond["match"].(map[string]any)
		if match["value"] != "acme" {
			t.Errorf("expected value 'acme', got %v", match["value"])
		}
	})

	t.Run("in", func(t *testing.T) {
		f := buildQdrantFilter(&dory.MetadataFilter{
			Field: "status",
			Op:    dory.FilterOpIn,
			Value: []string{"active", "pending"},
		})
		must, _ := f["must"].([]any)
		if len(must) != 1 {
			t.Fatalf("expected 1 must condition, got %d", len(must))
		}
		cond, _ := must[0].(map[string]any)
		match, _ := cond["match"].(map[string]any)
		anyVals, _ := match["any"].([]any)
		if len(anyVals) != 2 {
			t.Fatalf("expected 2 values, got %d", len(anyVals))
		}
	})

	t.Run("any_of", func(t *testing.T) {
		f := buildQdrantFilter(&dory.MetadataFilter{
			Field: "roles",
			Op:    dory.FilterOpAnyOf,
			Value: []string{"admin"},
		})
		must, _ := f["must"].([]any)
		if len(must) != 1 {
			t.Fatalf("expected 1 must condition, got %d", len(must))
		}
	})

	t.Run("unknown op", func(t *testing.T) {
		f := buildQdrantFilter(&dory.MetadataFilter{
			Field: "x",
			Op:    "unknown",
			Value: "y",
		})
		if f != nil {
			t.Errorf("expected nil for unknown op, got %v", f)
		}
	})
}
