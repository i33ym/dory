package dory

import "context"

// Subject identifies the entity making the retrieval request.
type Subject string

// Resource identifies a document or chunk for authorization purposes.
type Resource string

// Action describes what the subject wants to do with the resource.
type Action string

const (
	// ActionRead is the action checked for retrieval. Most RAG systems
	// only need this single action.
	ActionRead Action = "read"
)

// CheckRequest is the input to a single authorization check.
type CheckRequest struct {
	Subject  Subject
	Action   Action
	Resource Resource
}

// FilterRequest is the input to a bulk authorization filter.
type FilterRequest struct {
	Subject Subject
	Action  Action
	// Candidates, if non-nil, restricts the check to this set of resources.
	// If nil, implementations should return ALL authorized resources for
	// the subject — used for the pre-filter path.
	Candidates []Resource
}

// ResourceSet is the result of a FilterRequest.
type ResourceSet struct {
	// Resources is the explicit list of authorized resource IDs.
	Resources []Resource

	// Predicate, if non-nil, can be passed directly to a VectorStore
	// to restrict the search space at the database level.
	Predicate *MetadataFilter
}

// AuthorizationMode controls where in the pipeline authorization is enforced.
type AuthorizationMode int

const (
	// PostFilter retrieves candidates first, then authorizes each result.
	// This is the safe default: correct regardless of metadata staleness.
	PostFilter AuthorizationMode = iota

	// PreFilter passes authorization constraints to the VectorStore as
	// metadata filters before the similarity search runs.
	// Faster, but requires keeping chunk metadata in sync with the
	// authorization system when permissions change.
	PreFilter

	// Hybrid applies tenant isolation as a pre-filter and fine-grained
	// per-document authorization as a post-filter.
	Hybrid
)

// Authorizer is the authorization backend interface.
// OpenFGA, Casbin, simple allowlists, and the NoopAuthorizer all implement this.
type Authorizer interface {
	// Check answers: can this subject perform this action on this resource?
	Check(ctx context.Context, req CheckRequest) (bool, error)

	// Filter answers: which of these resources (or all resources if
	// Candidates is nil) can this subject access?
	Filter(ctx context.Context, req FilterRequest) (ResourceSet, error)
}
