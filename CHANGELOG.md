# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0-alpha.3] - 2026-04-06

### Added

#### Core
- `Content` interface abstracting text, binary, and streaming content
- `Document` constructor with validation, SHA-256 fingerprint, functional options
- `Pipeline` type wiring splitter, embedder, store, retriever, reranker, and authorizer
- `ScoreEntry` for full score provenance across pipeline stages
- `Position` type with byte offsets for chunk location tracking
- `UnitEnvelope` for cross-process serialization of `RetrievedUnit`
- `NewGraphFact` and `NewStructuredRow` constructors

#### Chunking
- `chunk.Fixed` — fixed-size with overlap
- `chunk.Recursive` — recursive character splitting by separator hierarchy
- `chunk.Sentence` — sentence-aware grouping with overlap
- `chunk.Semantic` — boundary detection via embedding similarity
- `chunk.Late` — late chunking with metadata marking
- `chunk.Contextual` — LLM-generated context prefixes
- `chunk.Proposition` — LLM-based atomic proposition extraction

#### Retrieval
- `retrieve.Vector` — dense vector similarity search
- `retrieve.BM25` — sparse keyword search with in-memory inverted index
- `retrieve.Hybrid` — Reciprocal Rank Fusion across multiple retrievers
- `retrieve.Ensemble` — multi-retriever concatenation with deduplication
- `retrieve.Router` — query routing to best-matching retriever
- `retrieve.Graph` — in-memory knowledge graph triple matching
- `retrieve.Structured` — Text-to-SQL retrieval
- `retrieve.Web` — web search retrieval

#### Reranking
- `rerank.CrossEncoder` — concurrent cross-encoder scoring with TopK and threshold
- `rerank.LostInTheMiddle` — reorders strongest results to context edges

#### Vector Stores
- `store.Memory` — in-memory brute-force cosine search
- `store.PgVector` — PostgreSQL + pgvector via `database/sql`
- `store.Qdrant` — Qdrant via HTTP REST API

#### Authorization
- `auth.NoopAuthorizer` — passthrough for development and testing
- `auth.Allowlist` — dynamic grant/revoke allowlist
- `auth.OpenFGA` — OpenFGA via HTTP REST API
- `auth.Casbin` — RBAC with role assignments and wildcards
- Pre-filter, post-filter, and hybrid authorization modes

#### Evaluation
- `eval.RetrieverEvaluator` — context precision, context recall
- LLM-as-judge scoring for faithfulness and answer relevance

#### Embedding
- `embed.OpenAI` — OpenAI embedding API via official Go SDK

#### Examples
- `basic_rag` — vector retrieval with OpenAI embeddings
- `hybrid_rag` — BM25 + vector with RRF fusion
- `graph_rag` — knowledge graph triple retrieval
- `with_auth` — allowlist authorization with post-filtering

## [0.1.0-alpha.1] - 2026-04-06

### Added
- Initial project scaffold with core interfaces and types
