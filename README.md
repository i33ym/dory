# Dory

[![Go Reference](https://pkg.go.dev/badge/github.com/i33ym/dory.svg)](https://pkg.go.dev/github.com/i33ym/dory)
[![Go Report Card](https://goreportcard.com/badge/github.com/i33ym/dory)](https://goreportcard.com/report/github.com/i33ym/dory)
[![CI](https://github.com/i33ym/dory/actions/workflows/ci.yml/badge.svg)](https://github.com/i33ym/dory/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

**Dory is a retrieval intelligence library for Go.**

LLMs are a bit like Dory — they know a lot, but they cannot remember what
you told them last Tuesday. This library fixes that.

Dory provides a modular, interface-driven pipeline for chunking, indexing,
retrieving, reranking, and evaluating knowledge over arbitrary backends,
with authorization built in from the ground up. It is the retrieval layer
the Go AI ecosystem has been missing.

## Design Philosophy

Dory has opinions about *algorithms* and no opinions about *infrastructure*.
It ships with implementations of proven retrieval techniques — semantic chunking,
hybrid BM25/vector search with RRF, cross-encoder reranking, small-to-big
retrieval, and more — but every stage of the pipeline is expressed as a Go
interface. Swap your vector store, your embedding model, or your authorization
backend without changing any pipeline code.

## Features

- **Seven chunking strategies** — fixed-size, recursive, sentence-aware, semantic
  boundary detection, late chunking, contextual retrieval, and proposition extraction
- **Six retrieval paradigms** — dense vector, sparse BM25, hybrid with RRF, graph,
  structured (Text-to-SQL), and web search — composable via ensemble and router retrievers
- **Reranking** — cross-encoder reranking and lost-in-the-middle reordering
- **Authorization built-in** — pre-filter, post-filter, and hybrid modes with
  pluggable backends (OpenFGA, Casbin, allowlist, or your own)
- **Evaluation pipeline** — context precision, context recall, faithfulness, and
  answer relevance — all computable without human annotation via LLM-as-judge
- **Zero infrastructure opinions** — bring your own vector store, embedder, and
  authorization system

## Installation

```bash
go get github.com/i33ym/dory
```

Requires Go 1.23 or later.

## Quick Start

```go
ctx := context.Background()

doc := &dory.Document{
    ID:       "doc-001",
    Content:  "Your document content here.",
    MimeType: "text/plain",
}

splitter := chunk.NewFixed(chunk.FixedConfig{Size: 512, Overlap: 64})
chunks, _ := splitter.Split(ctx, doc)

embedder := embed.NewOpenAI("text-embedding-3-small")
vectorStore := store.NewMemory()

// embed chunks and store them ...

retriever := retrieve.NewVector(vectorStore, embedder)
results, _ := retriever.Retrieve(ctx, dory.Query{
    Text: "What does this document say?",
    TopK: 5,
})

for _, unit := range results {
    fmt.Println(unit.AsText())
}
```

See the [examples/](examples/) directory for complete working examples covering
hybrid retrieval, graph retrieval, and authorization.

## Pipeline Overview

```
Document
   │
   ▼
Splitter ──────────── seven strategies
   │
   ▼
Embedder ──────────── any provider
   │
   ▼
VectorStore ────────── any backend
   │
   ▼
Retriever ──────────── six paradigms + ensemble + router
   │
   ▼
Reranker ───────────── cross-encoder, RRF, lost-in-the-middle
   │
   ▼
Authorizer ─────────── pre-filter / post-filter / hybrid
   │
   ▼
[]RetrievedUnit ─────── Chunk | GraphFact | StructuredRow
```

## Versioning

Dory follows [Semantic Versioning](https://semver.org). Release history is
maintained via annotated git tags. See [CHANGELOG.md](CHANGELOG.md) for
the full history of changes.

## Contributing

Contributions are welcome. Please open an issue before submitting a pull
request for significant changes, so the design can be discussed first.
See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

MIT. See [LICENSE](LICENSE).
