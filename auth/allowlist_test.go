package auth

import (
	"context"
	"sort"
	"testing"

	"github.com/i33ym/dory"
)

func TestAllowlist_Check(t *testing.T) {
	a := NewAllowlist(AllowlistConfig{
		Grants: map[string][]string{
			"alice": {"doc-1", "doc-2"},
			"bob":   {"doc-2", "doc-3"},
		},
	})
	ctx := context.Background()

	tests := []struct {
		name     string
		subject  string
		resource string
		want     bool
	}{
		{"allowed", "alice", "doc-1", true},
		{"allowed shared", "bob", "doc-2", true},
		{"denied", "alice", "doc-3", false},
		{"unknown subject", "charlie", "doc-1", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := a.Check(ctx, dory.CheckRequest{
				Subject:  dory.Subject(tt.subject),
				Action:   dory.ActionRead,
				Resource: dory.Resource(tt.resource),
			})
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("Check() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAllowlist_Filter_WithCandidates(t *testing.T) {
	a := NewAllowlist(AllowlistConfig{
		Grants: map[string][]string{
			"alice": {"doc-1", "doc-3"},
		},
	})
	ctx := context.Background()

	result, err := a.Filter(ctx, dory.FilterRequest{
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

func TestAllowlist_Filter_NilCandidates(t *testing.T) {
	a := NewAllowlist(AllowlistConfig{
		Grants: map[string][]string{
			"alice": {"doc-1", "doc-2"},
		},
	})
	ctx := context.Background()

	result, err := a.Filter(ctx, dory.FilterRequest{
		Subject: "alice",
		Action:  dory.ActionRead,
	})
	if err != nil {
		t.Fatal(err)
	}

	got := make([]string, len(result.Resources))
	for i, r := range result.Resources {
		got[i] = string(r)
	}
	sort.Strings(got)

	want := []string{"doc-1", "doc-2"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("resource %d: got %q, want %q", i, got[i], want[i])
		}
	}
}

func TestAllowlist_GrantRevoke(t *testing.T) {
	a := NewAllowlist(AllowlistConfig{})
	ctx := context.Background()

	// Initially denied.
	allowed, _ := a.Check(ctx, dory.CheckRequest{
		Subject: "alice", Action: dory.ActionRead, Resource: "doc-1",
	})
	if allowed {
		t.Fatal("expected denied before grant")
	}

	// Grant access.
	a.Grant("alice", "doc-1")
	allowed, _ = a.Check(ctx, dory.CheckRequest{
		Subject: "alice", Action: dory.ActionRead, Resource: "doc-1",
	})
	if !allowed {
		t.Fatal("expected allowed after grant")
	}

	// Revoke access.
	a.Revoke("alice", "doc-1")
	allowed, _ = a.Check(ctx, dory.CheckRequest{
		Subject: "alice", Action: dory.ActionRead, Resource: "doc-1",
	})
	if allowed {
		t.Fatal("expected denied after revoke")
	}
}
