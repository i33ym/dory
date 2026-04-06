package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/i33ym/dory"
)

func TestOpenFGA_NewValidation(t *testing.T) {
	_, err := NewOpenFGA(OpenFGAConfig{})
	if err == nil {
		t.Fatal("expected error for empty URL")
	}

	_, err = NewOpenFGA(OpenFGAConfig{URL: "http://localhost:8080"})
	if err == nil {
		t.Fatal("expected error for empty StoreID")
	}

	_, err = NewOpenFGA(OpenFGAConfig{URL: "http://localhost:8080", StoreID: "s1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOpenFGA_Check(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/check") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var body checkRequestBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}

		allowed := body.TupleKey.User == "alice" && body.TupleKey.Object == "doc-1"
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(checkResponse{Allowed: allowed})
	}))
	defer srv.Close()

	o, err := NewOpenFGA(OpenFGAConfig{
		URL:     srv.URL,
		StoreID: "store-1",
		ModelID: "model-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	allowed, err := o.Check(ctx, dory.CheckRequest{
		Subject: "alice", Action: dory.ActionRead, Resource: "doc-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !allowed {
		t.Error("expected allowed")
	}

	allowed, err = o.Check(ctx, dory.CheckRequest{
		Subject: "alice", Action: dory.ActionRead, Resource: "doc-99",
	})
	if err != nil {
		t.Fatal(err)
	}
	if allowed {
		t.Error("expected denied")
	}
}

func TestOpenFGA_Filter_WithCandidates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body checkRequestBody
		_ = json.NewDecoder(r.Body).Decode(&body)

		allowed := body.TupleKey.Object == "doc-1" || body.TupleKey.Object == "doc-3"
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(checkResponse{Allowed: allowed})
	}))
	defer srv.Close()

	o, _ := NewOpenFGA(OpenFGAConfig{URL: srv.URL, StoreID: "s1"})
	ctx := context.Background()

	result, err := o.Filter(ctx, dory.FilterRequest{
		Subject:    "alice",
		Action:     dory.ActionRead,
		Candidates: []dory.Resource{"doc-1", "doc-2", "doc-3"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Resources) != 2 {
		t.Fatalf("got %d resources, want 2", len(result.Resources))
	}
}

func TestOpenFGA_Filter_NilCandidates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/list-objects") {
			t.Errorf("expected list-objects endpoint, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(listObjectsResponse{
			Objects: []string{"document:doc-1", "document:doc-2"},
		})
	}))
	defer srv.Close()

	o, _ := NewOpenFGA(OpenFGAConfig{URL: srv.URL, StoreID: "s1"})
	ctx := context.Background()

	result, err := o.Filter(ctx, dory.FilterRequest{
		Subject: "alice",
		Action:  dory.ActionRead,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Resources) != 2 {
		t.Fatalf("got %d resources, want 2", len(result.Resources))
	}
}

func TestOpenFGA_APIKeyHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-key" {
			t.Errorf("got auth header %q, want %q", auth, "Bearer test-key")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(checkResponse{Allowed: true})
	}))
	defer srv.Close()

	o, _ := NewOpenFGA(OpenFGAConfig{URL: srv.URL, StoreID: "s1", APIKey: "test-key"})
	_, _ = o.Check(context.Background(), dory.CheckRequest{
		Subject: "alice", Action: dory.ActionRead, Resource: "doc-1",
	})
}
