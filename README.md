# Dory

[![Go Reference](https://pkg.go.dev/badge/github.com/i33ym/dory.svg)](https://pkg.go.dev/github.com/i33ym/dory)
[![CI](https://github.com/i33ym/dory/actions/workflows/ci.yml/badge.svg)](https://github.com/i33ym/dory/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

Dory is a retrieval library for Go. It provides a modular pipeline for
chunking, embedding, indexing, retrieving, reranking, and evaluating
knowledge — with authorization built in from the ground up.

Every stage of the pipeline is expressed as a Go interface. Bring your
own vector store, embedding model, and authorization backend.

## Installation

```bash
go get github.com/i33ym/dory
```

## Quick Start

```go
// Create a document.
doc, _ := dory.NewDocument("doc-001", dory.TextContent("Your content here.", "text/plain"))

// Split into chunks.
splitter := chunk.NewFixed(chunk.FixedConfig{Size: 512, Overlap: 64})
chunks, _ := splitter.Split(ctx, doc)

// Embed and store.
embedder := embed.NewOpenAI("text-embedding-3-small")
vectorStore := store.NewMemory()
for _, c := range chunks {
    c.Vector, _ = embedder.Embed(ctx, c.AsText())
}
vectorStore.Store(ctx, chunks)

// Retrieve.
retriever := retrieve.NewVector(vectorStore, embedder)
results, _ := retriever.Retrieve(ctx, dory.Query{Text: "your question", TopK: 5})
```

Or wire everything together with a `Pipeline`:

```go
pipe, _ := dory.NewPipeline(dory.PipelineConfig{
    Splitter:  chunk.NewFixed(chunk.FixedConfig{Size: 512, Overlap: 64}),
    Embedder:  embed.NewOpenAI("text-embedding-3-small"),
    Store:     store.NewMemory(),
    Retriever: retriever,
    Reranker:  rerank.NewLostInTheMiddle(),
})

pipe.Ingest(ctx, doc)
results, _ := pipe.Retrieve(ctx, dory.Query{Text: "your question", TopK: 5})
```

See [examples/](examples/) for hybrid retrieval, graph retrieval, and authorization demos.

## What's Included

### Chunking

| Strategy | Package |
| --- | --- |
| Fixed-size with overlap | `chunk.NewFixed` |
| Recursive character splitting | `chunk.NewRecursive` |
| Sentence-aware grouping | `chunk.NewSentence` |
| Semantic boundary detection | `chunk.NewSemantic` |
| Late chunking | `chunk.NewLate` |
| Contextual retrieval | `chunk.NewContextual` |
| Proposition extraction | `chunk.NewProposition` |

### Retrieval

| Strategy | Package |
| --- | --- |
| Dense vector search | `retrieve.NewVector` |
| BM25 sparse search | `retrieve.NewBM25` |
| Hybrid (RRF fusion) | `retrieve.NewHybrid` |
| Ensemble (multi-retriever) | `retrieve.NewEnsemble` |
| Query routing | `retrieve.NewRouter` |
| Knowledge graph | `retrieve.NewGraph` |
| Text-to-SQL | `retrieve.NewStructured` |
| Web search | `retrieve.NewWeb` |

### Reranking

| Strategy | Package |
| --- | --- |
| Cross-encoder scoring | `rerank.NewCrossEncoder` |
| Lost-in-the-middle reordering | `rerank.NewLostInTheMiddle` |

### Vector Stores

| Backend | Package |
| --- | --- |
| In-memory (dev/test) | `store.NewMemory` |
| PostgreSQL + pgvector | `store.NewPgVector` |
| Qdrant | `store.NewQdrant` |

### Authorization

| Backend | Package |
| --- | --- |
| No-op (allow all) | `auth.NoopAuthorizer` |
| Allowlist | `auth.NewAllowlist` |
| OpenFGA | `auth.NewOpenFGA` |
| Casbin-style RBAC | `auth.NewCasbin` |

Pre-filter, post-filter, and hybrid authorization modes are supported
via `PipelineConfig.AuthMode`.

### Evaluation

Context precision, context recall, faithfulness, and answer relevance.
Faithfulness and answer relevance use LLM-as-judge scoring via a
configurable `JudgeFunc`.

## Contributing

Contributions are welcome. Please open an issue before submitting a pull
request for significant changes. See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

MIT
