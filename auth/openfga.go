package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/i33ym/dory"
)

// OpenFGAConfig configures the OpenFGA authorizer.
type OpenFGAConfig struct {
	// URL is the base URL of the OpenFGA server (e.g. "http://localhost:8080").
	URL string
	// StoreID is the OpenFGA store identifier.
	StoreID string
	// ModelID is the authorization model identifier.
	ModelID string
	// APIKey is an optional bearer token for authentication.
	APIKey string
}

// OpenFGA is an authorizer that delegates to an OpenFGA server via its HTTP REST API.
type OpenFGA struct {
	cfg    OpenFGAConfig
	client *http.Client
}

// NewOpenFGA creates a new OpenFGA authorizer. It returns an error if required
// configuration fields are missing.
func NewOpenFGA(config OpenFGAConfig) (*OpenFGA, error) {
	if config.URL == "" {
		return nil, fmt.Errorf("auth/openfga: URL is required")
	}
	if config.StoreID == "" {
		return nil, fmt.Errorf("auth/openfga: StoreID is required")
	}
	return &OpenFGA{
		cfg:    config,
		client: &http.Client{},
	}, nil
}

// checkRequest is the JSON body for the OpenFGA Check API.
type checkRequestBody struct {
	TupleKey             tupleKey `json:"tuple_key"`
	AuthorizationModelID string   `json:"authorization_model_id,omitempty"`
}

type tupleKey struct {
	User     string `json:"user"`
	Relation string `json:"relation"`
	Object   string `json:"object"`
}

type checkResponse struct {
	Allowed bool `json:"allowed"`
}

type listObjectsRequestBody struct {
	AuthorizationModelID string `json:"authorization_model_id,omitempty"`
	User                 string `json:"user"`
	Relation             string `json:"relation"`
	Type                 string `json:"type"`
}

type listObjectsResponse struct {
	Objects []string `json:"objects"`
}

// Check asks the OpenFGA server whether the subject can perform the action on the resource.
func (o *OpenFGA) Check(ctx context.Context, req dory.CheckRequest) (bool, error) {
	body := checkRequestBody{
		TupleKey: tupleKey{
			User:     string(req.Subject),
			Relation: string(req.Action),
			Object:   string(req.Resource),
		},
		AuthorizationModelID: o.cfg.ModelID,
	}

	var resp checkResponse
	if err := o.post(ctx, fmt.Sprintf("/stores/%s/check", o.cfg.StoreID), body, &resp); err != nil {
		return false, err
	}
	return resp.Allowed, nil
}

// Filter returns the subset of candidates the subject may access. When
// Candidates is nil it uses the OpenFGA list-objects endpoint instead.
func (o *OpenFGA) Filter(ctx context.Context, req dory.FilterRequest) (dory.ResourceSet, error) {
	if req.Candidates != nil {
		var result []dory.Resource
		for _, c := range req.Candidates {
			allowed, err := o.Check(ctx, dory.CheckRequest{
				Subject:  req.Subject,
				Action:   req.Action,
				Resource: c,
			})
			if err != nil {
				return dory.ResourceSet{}, err
			}
			if allowed {
				result = append(result, c)
			}
		}
		return dory.ResourceSet{Resources: result}, nil
	}

	// No candidates — use list-objects.
	body := listObjectsRequestBody{
		AuthorizationModelID: o.cfg.ModelID,
		User:                 string(req.Subject),
		Relation:             string(req.Action),
		Type:                 "document",
	}

	var resp listObjectsResponse
	if err := o.post(ctx, fmt.Sprintf("/stores/%s/list-objects", o.cfg.StoreID), body, &resp); err != nil {
		return dory.ResourceSet{}, err
	}

	resources := make([]dory.Resource, len(resp.Objects))
	for i, obj := range resp.Objects {
		resources[i] = dory.Resource(obj)
	}
	return dory.ResourceSet{Resources: resources}, nil
}

func (o *OpenFGA) post(ctx context.Context, path string, body any, result any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("auth/openfga: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, o.cfg.URL+path, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("auth/openfga: create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if o.cfg.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+o.cfg.APIKey)
	}

	resp, err := o.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("auth/openfga: do request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("auth/openfga: unexpected status %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("auth/openfga: decode response: %w", err)
	}
	return nil
}
