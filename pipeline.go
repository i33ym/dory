package dory

import (
	"context"
	"fmt"
)

// PipelineConfig holds the components needed to construct a Pipeline.
// Splitter, Embedder, Store, and Retriever are required; Reranker and
// Authorizer are optional.
type PipelineConfig struct {
	Splitter  Splitter
	Embedder  Embedder
	Store     VectorStore
	Retriever Retriever

	// Reranker, if non-nil, reorders retrieval results for higher precision.
	Reranker Reranker

	// Authorizer, if non-nil, enforces access control on retrieval results.
	Authorizer Authorizer

	// AuthMode controls where authorization is enforced. Defaults to PostFilter.
	// Hybrid mode is not yet implemented and falls back to PostFilter.
	AuthMode AuthorizationMode

	// Hooks are called at key points in the pipeline lifecycle.
	// Multiple hooks are called in the order they appear in the slice.
	Hooks []Hook
}

// Pipeline wires Dory's pipeline stages together into a single coherent
// retrieval flow: ingest documents, retrieve relevant units, and
// optionally rerank and authorize the results.
type Pipeline struct {
	splitter   Splitter
	embedder   Embedder
	store      VectorStore
	retriever  Retriever
	reranker   Reranker
	authorizer Authorizer
	authMode   AuthorizationMode
	hooks      []Hook
}

// NewPipeline constructs a Pipeline from the given configuration.
// Returns an error if any required component is nil.
func NewPipeline(config PipelineConfig) (*Pipeline, error) {
	if config.Splitter == nil {
		return nil, fmt.Errorf("dory: pipeline requires a Splitter")
	}
	if config.Embedder == nil {
		return nil, fmt.Errorf("dory: pipeline requires an Embedder")
	}
	if config.Store == nil {
		return nil, fmt.Errorf("dory: pipeline requires a VectorStore")
	}
	if config.Retriever == nil {
		return nil, fmt.Errorf("dory: pipeline requires a Retriever")
	}

	return &Pipeline{
		splitter:   config.Splitter,
		embedder:   config.Embedder,
		store:      config.Store,
		retriever:  config.Retriever,
		reranker:   config.Reranker,
		authorizer: config.Authorizer,
		authMode:   config.AuthMode,
		hooks:      config.Hooks,
	}, nil
}

// Ingest splits each document into chunks, embeds them in batch, and
// stores them in the vector store. This is the ingestion path.
func (p *Pipeline) Ingest(ctx context.Context, docs ...*Document) error {
	for _, h := range p.hooks {
		if h.BeforeIngest != nil {
			h.BeforeIngest(ctx, len(docs))
		}
	}

	var totalChunks int
	for _, doc := range docs {
		chunks, err := p.splitter.Split(ctx, doc)
		if err != nil {
			err = fmt.Errorf("dory: split document %s: %w", doc.ID(), err)
			for _, h := range p.hooks {
				if h.AfterIngest != nil {
					h.AfterIngest(ctx, totalChunks, err)
				}
			}
			return err
		}
		if len(chunks) == 0 {
			continue
		}

		// Collect texts for batch embedding.
		texts := make([]string, len(chunks))
		for i, c := range chunks {
			texts[i] = c.Text()
		}

		vectors, err := p.embedder.EmbedBatch(ctx, texts)
		if err != nil {
			err = fmt.Errorf("dory: embed chunks for document %s: %w", doc.ID(), err)
			for _, h := range p.hooks {
				if h.AfterIngest != nil {
					h.AfterIngest(ctx, totalChunks, err)
				}
			}
			return err
		}

		for i, c := range chunks {
			c.Vector = vectors[i]
		}

		if err := p.store.Store(ctx, chunks); err != nil {
			err = fmt.Errorf("dory: store chunks for document %s: %w", doc.ID(), err)
			for _, h := range p.hooks {
				if h.AfterIngest != nil {
					h.AfterIngest(ctx, totalChunks, err)
				}
			}
			return err
		}

		totalChunks += len(chunks)
	}

	for _, h := range p.hooks {
		if h.AfterIngest != nil {
			h.AfterIngest(ctx, totalChunks, nil)
		}
	}
	return nil
}

// Retrieve finds relevant units for the given query. It calls the
// retriever, then optionally reranks, then optionally authorizes
// based on the configured AuthorizationMode.
func (p *Pipeline) Retrieve(ctx context.Context, q Query) ([]RetrievedUnit, error) {
	for _, h := range p.hooks {
		if h.BeforeRetrieve != nil {
			h.BeforeRetrieve(ctx, q)
		}
	}

	// PreFilter: call Authorizer.Filter before retrieval to restrict search space.
	if p.authorizer != nil && p.authMode == PreFilter {
		rs, err := p.authorizer.Filter(ctx, FilterRequest{
			Subject: Subject(q.Subject),
			Action:  ActionRead,
		})
		if err != nil {
			err = fmt.Errorf("dory: pre-filter authorization: %w", err)
			for _, h := range p.hooks {
				if h.AfterRetrieve != nil {
					h.AfterRetrieve(ctx, 0, err)
				}
			}
			return nil, err
		}
		if rs.Predicate != nil {
			q.Filters = append(q.Filters, *rs.Predicate)
		}
	}

	units, err := p.retriever.Retrieve(ctx, q)
	if err != nil {
		err = fmt.Errorf("dory: retrieve: %w", err)
		for _, h := range p.hooks {
			if h.AfterRetrieve != nil {
				h.AfterRetrieve(ctx, 0, err)
			}
		}
		return nil, err
	}

	// Rerank if a reranker is configured.
	if p.reranker != nil {
		for _, h := range p.hooks {
			if h.BeforeRerank != nil {
				h.BeforeRerank(ctx, q.Text, len(units))
			}
		}

		units, err = p.reranker.Rerank(ctx, q.Text, units)
		if err != nil {
			err = fmt.Errorf("dory: rerank: %w", err)
			for _, h := range p.hooks {
				if h.AfterRerank != nil {
					h.AfterRerank(ctx, 0, err)
				}
			}
			for _, h := range p.hooks {
				if h.AfterRetrieve != nil {
					h.AfterRetrieve(ctx, 0, err)
				}
			}
			return nil, err
		}

		for _, h := range p.hooks {
			if h.AfterRerank != nil {
				h.AfterRerank(ctx, len(units), nil)
			}
		}
	}

	// PostFilter (or Hybrid, which falls back to PostFilter for now):
	// filter each result with Authorizer.Check after retrieval.
	if p.authorizer != nil && (p.authMode == PostFilter || p.authMode == Hybrid) {
		var allowed []RetrievedUnit
		for _, u := range units {
			ok, err := p.authorizer.Check(ctx, CheckRequest{
				Subject:  Subject(q.Subject),
				Action:   ActionRead,
				Resource: Resource(u.SourceDocumentID()),
			})
			if err != nil {
				err = fmt.Errorf("dory: post-filter authorization: %w", err)
				for _, h := range p.hooks {
					if h.AfterRetrieve != nil {
						h.AfterRetrieve(ctx, 0, err)
					}
				}
				return nil, err
			}
			if ok {
				allowed = append(allowed, u)
			}
		}
		units = allowed
	}

	for _, h := range p.hooks {
		if h.AfterRetrieve != nil {
			h.AfterRetrieve(ctx, len(units), nil)
		}
	}
	return units, nil
}

// Delete removes chunks associated with the given document IDs from the
// vector store.
func (p *Pipeline) Delete(ctx context.Context, docIDs ...string) error {
	return p.store.Delete(ctx, docIDs)
}
