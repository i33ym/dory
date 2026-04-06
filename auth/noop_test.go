package auth

import (
	"context"
	"testing"

	"github.com/i33ym/dory"
)

func TestNoopAuthorizer_Check(t *testing.T) {
	a := &NoopAuthorizer{}
	ctx := context.Background()

	allowed, err := a.Check(ctx, dory.CheckRequest{
		Subject:  "user-1",
		Action:   dory.ActionRead,
		Resource: "doc-secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !allowed {
		t.Error("NoopAuthorizer should always allow")
	}
}

func TestNoopAuthorizer_Filter_WithCandidates(t *testing.T) {
	a := &NoopAuthorizer{}
	ctx := context.Background()

	candidates := []dory.Resource{"doc-1", "doc-2", "doc-3"}
	result, err := a.Filter(ctx, dory.FilterRequest{
		Subject:    "user-1",
		Action:     dory.ActionRead,
		Candidates: candidates,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Resources) != len(candidates) {
		t.Fatalf("got %d resources, want %d", len(result.Resources), len(candidates))
	}
	for i, r := range result.Resources {
		if r != candidates[i] {
			t.Errorf("resource %d: got %q, want %q", i, r, candidates[i])
		}
	}
}

func TestNoopAuthorizer_Filter_NilCandidates(t *testing.T) {
	a := &NoopAuthorizer{}
	ctx := context.Background()

	result, err := a.Filter(ctx, dory.FilterRequest{
		Subject: "user-1",
		Action:  dory.ActionRead,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Resources != nil {
		t.Errorf("expected nil resources for nil candidates, got %v", result.Resources)
	}
}
