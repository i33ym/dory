// Package dory provides a retrieval intelligence library for Go.
//
// Dory is organized around a pipeline of composable, interface-driven stages:
//
//  1. Chunking    — split documents into retrievable units
//  2. Embedding   — transform text into vector representations
//  3. Indexing    — store chunks in a searchable backend
//  4. Retrieval   — find the most relevant units for a query
//  5. Reranking   — reorder candidates by cross-encoder relevance
//  6. Authorization — filter results by what the caller is allowed to see
//  7. Evaluation  — measure retrieval quality with quantitative metrics
//
// Every stage is expressed as a Go interface. Dory ships with concrete
// implementations for each, but any implementation of the interface works —
// the library has no opinion about which vector store, embedding model,
// or authorization backend you use.
//
// The canonical entry point for most users is [Pipeline], which wires
// the stages together into a single coherent retrieval flow.
package dory
