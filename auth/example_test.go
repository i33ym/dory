package auth_test

import (
	"context"
	"fmt"

	"github.com/i33ym/dory"
	"github.com/i33ym/dory/auth"
)

func ExampleNewAllowlist() {
	az := auth.NewAllowlist(auth.AllowlistConfig{
		Grants: map[string][]string{
			"alice": {"doc-1", "doc-2"},
			"bob":   {"doc-2"},
		},
	})

	ctx := context.Background()

	allowed, _ := az.Check(ctx, dory.CheckRequest{
		Subject:  "alice",
		Resource: "doc-1",
	})
	fmt.Println("alice -> doc-1:", allowed)

	denied, _ := az.Check(ctx, dory.CheckRequest{
		Subject:  "bob",
		Resource: "doc-1",
	})
	fmt.Println("bob -> doc-1:", denied)
	// Output:
	// alice -> doc-1: true
	// bob -> doc-1: false
}
