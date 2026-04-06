package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/i33ym/dory"
)

// PgVectorConfig holds configuration for a PostgreSQL + pgvector store.
type PgVectorConfig struct {
	// DB is the database connection pool. The caller is responsible for
	// opening and closing it. Must have the pgvector extension installed.
	DB *sql.DB

	// TableName is the name of the table to store chunks in.
	// Defaults to "dory_chunks" if empty.
	TableName string

	// Dimensions is the size of the embedding vectors (e.g. 1536).
	Dimensions int
}

// PgVector is a VectorStore backed by PostgreSQL with the pgvector extension.
type PgVector struct {
	db         *sql.DB
	table      string
	dimensions int
}

// NewPgVector creates a new pgvector-backed vector store.
func NewPgVector(config PgVectorConfig) (*PgVector, error) {
	if config.DB == nil {
		return nil, fmt.Errorf("dory/store: PgVectorConfig.DB must not be nil")
	}
	if config.Dimensions <= 0 {
		return nil, fmt.Errorf("dory/store: PgVectorConfig.Dimensions must be > 0")
	}
	table := config.TableName
	if table == "" {
		table = "dory_chunks"
	}
	return &PgVector{
		db:         config.DB,
		table:      table,
		dimensions: config.Dimensions,
	}, nil
}

// EnsureTable creates the chunks table and pgvector extension if they do not exist.
func (p *PgVector) EnsureTable(ctx context.Context) error {
	query := fmt.Sprintf(`
		CREATE EXTENSION IF NOT EXISTS vector;
		CREATE TABLE IF NOT EXISTS %s (
			id            TEXT PRIMARY KEY,
			source_doc_id TEXT NOT NULL,
			source_uri    TEXT NOT NULL DEFAULT '',
			content       TEXT NOT NULL,
			vector        vector(%d),
			metadata      JSONB DEFAULT '{}'
		);
	`, p.table, p.dimensions)
	_, err := p.db.ExecContext(ctx, query)
	return err
}

// Store upserts chunks into the pgvector table.
func (p *PgVector) Store(ctx context.Context, chunks []*dory.Chunk) error {
	if len(chunks) == 0 {
		return nil
	}

	query := fmt.Sprintf(`
		INSERT INTO %s (id, source_doc_id, source_uri, content, vector, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET
			source_doc_id = EXCLUDED.source_doc_id,
			source_uri    = EXCLUDED.source_uri,
			content       = EXCLUDED.content,
			vector        = EXCLUDED.vector,
			metadata      = EXCLUDED.metadata
	`, p.table)

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("dory/store: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("dory/store: prepare: %w", err)
	}
	defer func() { _ = stmt.Close() }()

	for _, c := range chunks {
		metaJSON, err := json.Marshal(c.Metadata())
		if err != nil {
			return fmt.Errorf("dory/store: marshal metadata for chunk %s: %w", c.ID(), err)
		}

		vecStr := pgvectorString(c.Vector)

		_, err = stmt.ExecContext(ctx, c.ID(), c.SourceDocumentID(), c.SourceURI(), c.AsText(), vecStr, string(metaJSON))
		if err != nil {
			return fmt.Errorf("dory/store: upsert chunk %s: %w", c.ID(), err)
		}
	}

	return tx.Commit()
}

// Search finds the top-k chunks by cosine similarity.
func (p *PgVector) Search(ctx context.Context, req dory.SearchRequest) ([]dory.ScoredChunk, error) {
	topK := req.TopK
	if topK <= 0 {
		topK = 10
	}

	whereClause, args := p.buildFilter(req.Filter, 2) // $1 is the query vector

	query := fmt.Sprintf(`
		SELECT id, source_doc_id, source_uri, content, metadata,
		       1 - (vector <=> $1) AS score
		FROM %s
		%s
		ORDER BY vector <=> $1
		LIMIT %d
	`, p.table, whereClause, topK)

	vecStr := pgvectorString(req.QueryVector)
	allArgs := append([]any{vecStr}, args...)

	rows, err := p.db.QueryContext(ctx, query, allArgs...)
	if err != nil {
		return nil, fmt.Errorf("dory/store: search query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var results []dory.ScoredChunk
	for rows.Next() {
		var (
			id, sourceDocID, sourceURI, content string
			metaJSON                            string
			score                               float64
		)
		if err := rows.Scan(&id, &sourceDocID, &sourceURI, &content, &metaJSON, &score); err != nil {
			return nil, fmt.Errorf("dory/store: scan row: %w", err)
		}

		var meta map[string]any
		if metaJSON != "" {
			if err := json.Unmarshal([]byte(metaJSON), &meta); err != nil {
				return nil, fmt.Errorf("dory/store: unmarshal metadata: %w", err)
			}
		}

		chunk := dory.NewChunk(id, sourceDocID, content, meta)
		results = append(results, dory.ScoredChunk{Chunk: chunk, Score: score})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dory/store: rows iteration: %w", err)
	}

	return results, nil
}

// Delete removes chunks by their IDs.
func (p *PgVector) Delete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	query := fmt.Sprintf(`DELETE FROM %s WHERE id = ANY($1)`, p.table)
	_, err := p.db.ExecContext(ctx, query, pgStringArray(ids))
	return err
}

// buildFilter translates a MetadataFilter into a SQL WHERE clause with
// parameterized arguments. paramOffset is the starting parameter number.
func (p *PgVector) buildFilter(f *dory.MetadataFilter, paramOffset int) (string, []any) {
	if f == nil {
		return "", nil
	}

	switch f.Op {
	case dory.FilterOpEq:
		param := fmt.Sprintf("$%d", paramOffset)
		clause := fmt.Sprintf("WHERE metadata @> %s::jsonb", param)
		filterJSON, _ := json.Marshal(map[string]any{f.Field: f.Value})
		return clause, []any{string(filterJSON)}

	case dory.FilterOpIn:
		// metadata->>'field' IN (values...)
		// Use jsonb: metadata->>'field' = ANY($N)
		param := fmt.Sprintf("$%d", paramOffset)
		fieldParam := fmt.Sprintf("$%d", paramOffset+1)
		clause := fmt.Sprintf("WHERE metadata->>%s = ANY(%s)", fieldParam, param)
		vals, _ := f.Value.([]string)
		return clause, []any{pgStringArray(vals), f.Field}

	case dory.FilterOpAnyOf:
		// Check if a jsonb array field overlaps with the given values.
		// metadata->'field' ?| array[values]
		param := fmt.Sprintf("$%d", paramOffset)
		fieldParam := fmt.Sprintf("$%d", paramOffset+1)
		clause := fmt.Sprintf("WHERE metadata->%s ?| %s", fieldParam, param)
		vals, _ := f.Value.([]string)
		return clause, []any{pgStringArray(vals), f.Field}

	default:
		return "", nil
	}
}

// pgvectorString formats a float32 slice as a pgvector literal: "[1.0,2.0,3.0]".
func pgvectorString(v []float32) string {
	if len(v) == 0 {
		return "[]"
	}
	var b strings.Builder
	b.WriteByte('[')
	for i, f := range v {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, "%g", f)
	}
	b.WriteByte(']')
	return b.String()
}

// pgStringArray formats a string slice as a PostgreSQL array literal.
func pgStringArray(ss []string) string {
	parts := make([]string, len(ss))
	for i, s := range ss {
		// Escape backslashes and double quotes for PostgreSQL array elements.
		escaped := strings.ReplaceAll(s, `\`, `\\`)
		escaped = strings.ReplaceAll(escaped, `"`, `\"`)
		parts[i] = `"` + escaped + `"`
	}
	return "{" + strings.Join(parts, ",") + "}"
}
