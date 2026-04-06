package auth

import (
	"context"
	"testing"

	"github.com/i33ym/dory"
)

func TestCasbin_DirectPolicy(t *testing.T) {
	c := NewCasbin(CasbinConfig{
		Policies: []Policy{
			{Subject: "alice", Resource: "doc-1", Action: "read"},
			{Subject: "bob", Resource: "doc-2", Action: "read"},
		},
	})
	ctx := context.Background()

	tests := []struct {
		name     string
		subject  string
		resource string
		action   string
		want     bool
	}{
		{"alice can read doc-1", "alice", "doc-1", "read", true},
		{"alice cannot read doc-2", "alice", "doc-2", "read", false},
		{"bob can read doc-2", "bob", "doc-2", "read", true},
		{"bob cannot write doc-2", "bob", "doc-2", "write", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := c.Check(ctx, dory.CheckRequest{
				Subject:  dory.Subject(tt.subject),
				Action:   dory.Action(tt.action),
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

func TestCasbin_RoleBasedPolicy(t *testing.T) {
	c := NewCasbin(CasbinConfig{
		Policies: []Policy{
			{Subject: "admin", Resource: "doc-secret", Action: "read"},
		},
		RoleAssignments: map[string][]string{
			"alice": {"admin"},
		},
	})
	ctx := context.Background()

	// alice has admin role -> can read doc-secret.
	allowed, err := c.Check(ctx, dory.CheckRequest{
		Subject: "alice", Action: "read", Resource: "doc-secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !allowed {
		t.Error("alice should be allowed via admin role")
	}

	// bob has no roles -> denied.
	allowed, err = c.Check(ctx, dory.CheckRequest{
		Subject: "bob", Action: "read", Resource: "doc-secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	if allowed {
		t.Error("bob should be denied")
	}
}

func TestCasbin_Wildcards(t *testing.T) {
	c := NewCasbin(CasbinConfig{
		Policies: []Policy{
			{Subject: "admin", Resource: "*", Action: "*"},
			{Subject: "reader", Resource: "*", Action: "read"},
		},
		RoleAssignments: map[string][]string{
			"alice": {"admin"},
			"bob":   {"reader"},
		},
	})
	ctx := context.Background()

	// admin can do anything.
	allowed, _ := c.Check(ctx, dory.CheckRequest{
		Subject: "alice", Action: "write", Resource: "doc-1",
	})
	if !allowed {
		t.Error("admin should be able to write")
	}

	// reader can read anything.
	allowed, _ = c.Check(ctx, dory.CheckRequest{
		Subject: "bob", Action: "read", Resource: "doc-99",
	})
	if !allowed {
		t.Error("reader should be able to read anything")
	}

	// reader cannot write.
	allowed, _ = c.Check(ctx, dory.CheckRequest{
		Subject: "bob", Action: "write", Resource: "doc-1",
	})
	if allowed {
		t.Error("reader should not be able to write")
	}
}

func TestCasbin_Filter(t *testing.T) {
	c := NewCasbin(CasbinConfig{
		Policies: []Policy{
			{Subject: "alice", Resource: "doc-1", Action: "read"},
			{Subject: "alice", Resource: "doc-3", Action: "read"},
		},
	})
	ctx := context.Background()

	result, err := c.Filter(ctx, dory.FilterRequest{
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

func TestCasbin_Filter_NilCandidates(t *testing.T) {
	c := NewCasbin(CasbinConfig{
		Policies: []Policy{
			{Subject: "alice", Resource: "doc-1", Action: "read"},
		},
	})
	ctx := context.Background()

	result, err := c.Filter(ctx, dory.FilterRequest{
		Subject: "alice",
		Action:  dory.ActionRead,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Resources) != 0 {
		t.Errorf("expected empty result for nil candidates, got %v", result.Resources)
	}
}
