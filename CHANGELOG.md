# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Core interfaces: `RetrievedUnit`, `Splitter`, `Embedder`, `VectorStore`,
  `Retriever`, `Reranker`, `Authorizer`, `Evaluator`
- Core types: `Chunk`, `GraphFact`, `StructuredRow`, `Document`
- `auth.NoopAuthorizer` — passthrough authorizer for development and testing
- `internal/similarity` — cosine similarity utility
- Examples: `basic_rag`, `hybrid_rag`, `graph_rag`, `with_auth`
