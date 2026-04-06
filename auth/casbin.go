package auth

import (
	"context"

	"github.com/i33ym/dory"
)

// Policy defines a single authorization rule in the Casbin-style authorizer.
type Policy struct {
	Subject  string
	Resource string
	Action   string
}

// CasbinConfig configures the Casbin-style authorizer.
type CasbinConfig struct {
	// Policies is the set of authorization rules.
	Policies []Policy
	// RoleAssignments maps a user to its assigned roles. When checking
	// authorization the user's roles are also matched against policies.
	RoleAssignments map[string][]string
}

// Casbin is a simplified policy-based authorizer following the Casbin RBAC model.
// It supports wildcard "*" matching for resources and actions, and role-based
// policy lookup via RoleAssignments.
type Casbin struct {
	policies        []Policy
	roleAssignments map[string][]string
}

// NewCasbin creates a Casbin authorizer from the given configuration.
func NewCasbin(config CasbinConfig) *Casbin {
	ra := config.RoleAssignments
	if ra == nil {
		ra = make(map[string][]string)
	}
	policies := make([]Policy, len(config.Policies))
	copy(policies, config.Policies)
	return &Casbin{
		policies:        policies,
		roleAssignments: ra,
	}
}

// Check returns true if the subject (or any of its roles) has a matching policy.
func (c *Casbin) Check(_ context.Context, req dory.CheckRequest) (bool, error) {
	subjects := c.effectiveSubjects(string(req.Subject))
	for _, p := range c.policies {
		if c.matchSubject(p.Subject, subjects) &&
			matchWildcard(p.Resource, string(req.Resource)) &&
			matchWildcard(p.Action, string(req.Action)) {
			return true, nil
		}
	}
	return false, nil
}

// Filter returns the subset of candidates the subject is allowed to access.
// If Candidates is nil an empty set is returned (pre-filter without candidates
// is not supported by this simple implementation).
func (c *Casbin) Filter(ctx context.Context, req dory.FilterRequest) (dory.ResourceSet, error) {
	if req.Candidates == nil {
		return dory.ResourceSet{}, nil
	}

	var result []dory.Resource
	for _, cand := range req.Candidates {
		allowed, err := c.Check(ctx, dory.CheckRequest{
			Subject:  req.Subject,
			Action:   req.Action,
			Resource: cand,
		})
		if err != nil {
			return dory.ResourceSet{}, err
		}
		if allowed {
			result = append(result, cand)
		}
	}
	return dory.ResourceSet{Resources: result}, nil
}

// effectiveSubjects returns the subject itself plus all roles assigned to it.
func (c *Casbin) effectiveSubjects(subject string) []string {
	subjects := []string{subject}
	if roles, ok := c.roleAssignments[subject]; ok {
		subjects = append(subjects, roles...)
	}
	return subjects
}

func (c *Casbin) matchSubject(policySub string, subjects []string) bool {
	for _, s := range subjects {
		if policySub == "*" || policySub == s {
			return true
		}
	}
	return false
}

func matchWildcard(pattern, value string) bool {
	return pattern == "*" || pattern == value
}
