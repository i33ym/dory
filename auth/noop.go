// Package auth provides authorization backend implementations for Dory.
// Each implementation satisfies the [dory.Authorizer] interface.
//
// The NoopAuthorizer allows all requests unconditionally and is suitable
// for development, testing, and public knowledge bases with no access control.
package auth

import (
	"context"

	"github.com/i33ym/dory"
)

// NoopAuthorizer permits all authorization checks unconditionally.
// Use this during development or when authorization is not required.
type NoopAuthorizer struct{}

func (n *NoopAuthorizer) Check(_ context.Context, _ dory.CheckRequest) (bool, error) {
	return true, nil
}

func (n *NoopAuthorizer) Filter(_ context.Context, req dory.FilterRequest) (dory.ResourceSet, error) {
	return dory.ResourceSet{Resources: req.Candidates}, nil
}
