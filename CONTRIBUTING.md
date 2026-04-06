# Contributing to Dory

Thank you for your interest in contributing.

## Before You Submit a Pull Request

For significant changes — new retrieval strategies, new chunking algorithms,
interface modifications — please open an issue first to discuss the design.
Dory's public interfaces are stable commitments, and changes require careful
consideration.

For bug fixes and documentation improvements, pull requests are welcome directly.

## Code Style

Follow standard Go conventions. Run `gofmt` and `go vet` before committing.
All exported types and functions must have documentation comments.
Avoid unnecessary abstraction — if a simpler design works, prefer it.

## Testing

All new functionality must include tests. Run the full test suite with:

    make test

## Commit Messages

Use conventional commit format:

    feat: add semantic chunking strategy
    fix: correct cosine similarity for zero vectors
    docs: add graph_rag example
    chore: update CI to Go 1.23

## Versioning

Dory uses semantic versioning. Maintainers tag releases. Contributors do not
need to update version numbers or modify CHANGELOG.md — that is done at release time.
