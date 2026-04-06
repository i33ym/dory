package auth

import (
	"context"
	"sync"

	"github.com/i33ym/dory"
)

// AllowlistConfig configures the allowlist authorizer.
type AllowlistConfig struct {
	// Grants maps a subject string to the list of resource strings it may access.
	Grants map[string][]string
}

// Allowlist is a simple authorizer backed by an in-memory map of
// subject -> allowed resources. It is safe for concurrent use.
type Allowlist struct {
	mu     sync.RWMutex
	grants map[string]map[string]struct{}
}

// NewAllowlist creates an Allowlist from the given configuration.
func NewAllowlist(config AllowlistConfig) *Allowlist {
	grants := make(map[string]map[string]struct{}, len(config.Grants))
	for sub, resources := range config.Grants {
		set := make(map[string]struct{}, len(resources))
		for _, r := range resources {
			set[r] = struct{}{}
		}
		grants[sub] = set
	}
	return &Allowlist{grants: grants}
}

// Check returns true if the subject has the resource in its grant list.
func (a *Allowlist) Check(_ context.Context, req dory.CheckRequest) (bool, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	resources, ok := a.grants[string(req.Subject)]
	if !ok {
		return false, nil
	}
	_, allowed := resources[string(req.Resource)]
	return allowed, nil
}

// Filter returns the subset of candidates the subject is allowed to access.
// If Candidates is nil, all resources granted to the subject are returned.
func (a *Allowlist) Filter(_ context.Context, req dory.FilterRequest) (dory.ResourceSet, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	resources := a.grants[string(req.Subject)]

	if req.Candidates == nil {
		// Return all granted resources.
		result := make([]dory.Resource, 0, len(resources))
		for r := range resources {
			result = append(result, dory.Resource(r))
		}
		return dory.ResourceSet{Resources: result}, nil
	}

	// Filter candidates.
	var result []dory.Resource
	for _, c := range req.Candidates {
		if _, ok := resources[string(c)]; ok {
			result = append(result, c)
		}
	}
	return dory.ResourceSet{Resources: result}, nil
}

// Grant adds a resource to a subject's allowlist.
func (a *Allowlist) Grant(subject, resource string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.grants[subject] == nil {
		a.grants[subject] = make(map[string]struct{})
	}
	a.grants[subject][resource] = struct{}{}
}

// Revoke removes a resource from a subject's allowlist.
func (a *Allowlist) Revoke(subject, resource string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if resources, ok := a.grants[subject]; ok {
		delete(resources, resource)
	}
}
