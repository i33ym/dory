package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/i33ym/dory"
	"github.com/i33ym/dory/internal/filter"
)

// SQLiteConfig holds the configuration for a SQLite vector store.
type SQLiteConfig struct {
	// DB is the open database/sql handle. The caller is responsible
	// for importing a driver (e.g. modernc.org/sqlite) and opening
	// the connection before passing it here.
	DB *sql.DB

	// TableName is the name of the table used for chunk storage.
	// Defaults to "dory_chunks" if empty.
	TableName string
}

// SQLite is a VectorStore backed by a SQLite database.
// Vectors are stored as JSON arrays and cosine similarity is
// computed in Go (brute-force), similar to the Memory store
// but with persistence.
type SQLite struct {
	db        *sql.DB
	tableName string
}

// NewSQLite creates a new SQLite vector store.
// It validates the config but does not create the table —
// call EnsureTable to do that.
func NewSQLite(config SQLiteConfig) (*SQLite, error) {
	if config.DB == nil {
		return nil, fmt.Errorf("store: SQLiteConfig.DB must not be nil")
	}
	tableName := config.TableName
	if tableName == "" {
		tableName = "dory_chunks"
	}
	return &SQLite{
		db:        config.DB,
		tableName: tableName,
	}, nil
}

// EnsureTable creates the chunks table if it does not exist.
func (s *SQLite) EnsureTable(ctx context.Context) error {
	query := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		id            TEXT PRIMARY KEY,
		source_doc_id TEXT,
		source_uri    TEXT,
		content       TEXT,
		vector        TEXT,
		metadata      TEXT
	)`, s.tableName)

	_, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("store: create table: %w", err)
	}
	return nil
}

// Store persists chunks into SQLite.
func (s *SQLite) Store(ctx context.Context, chunks []*dory.Chunk) error {
	query := fmt.Sprintf(
		`INSERT OR REPLACE INTO %s (id, source_doc_id, source_uri, content, vector, metadata) VALUES (?, ?, ?, ?, ?, ?)`,
		s.tableName,
	)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("store: prepare: %w", err)
	}
	defer func() { _ = stmt.Close() }()

	for _, c := range chunks {
		vecJSON, err := marshalVector(c.Vector)
		if err != nil {
			return fmt.Errorf("store: marshal vector for %s: %w", c.ID(), err)
		}
		metaJSON, err := marshalMetadata(c.Metadata())
		if err != nil {
			return fmt.Errorf("store: marshal metadata for %s: %w", c.ID(), err)
		}

		_, err = stmt.ExecContext(ctx, c.ID(), c.SourceDocumentID(), c.SourceURI(), c.AsText(), vecJSON, metaJSON)
		if err != nil {
			return fmt.Errorf("store: insert %s: %w", c.ID(), err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: commit: %w", err)
	}
	return nil
}

// Search finds the top-k chunks by cosine similarity, applying metadata filters.
func (s *SQLite) Search(ctx context.Context, req dory.SearchRequest) ([]dory.ScoredChunk, error) {
	query := fmt.Sprintf(`SELECT id, source_doc_id, source_uri, content, vector, metadata FROM %s`, s.tableName)

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("store: query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	type scored struct {
		chunk *dory.Chunk
		score float64
	}
	var results []scored

	for rows.Next() {
		var id, sourceDocID, sourceURI, content, vecJSON, metaJSON string
		if err := rows.Scan(&id, &sourceDocID, &sourceURI, &content, &vecJSON, &metaJSON); err != nil {
			return nil, fmt.Errorf("store: scan row: %w", err)
		}

		vec, err := unmarshalVector(vecJSON)
		if err != nil {
			continue // skip rows with invalid vectors
		}

		meta, err := unmarshalMetadata(metaJSON)
		if err != nil {
			meta = nil
		}

		c := dory.NewChunk(id, sourceDocID, content, meta)
		c.Vector = vec

		if req.Filter != nil && !filter.Match(c.Metadata(), req.Filter) {
			continue
		}

		if len(req.QueryVector) == 0 {
			continue
		}

		score := cosineSimilarity(req.QueryVector, vec)
		results = append(results, scored{chunk: c, score: score})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: rows iteration: %w", err)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	topK := req.TopK
	if topK > len(results) {
		topK = len(results)
	}

	out := make([]dory.ScoredChunk, topK)
	for i := 0; i < topK; i++ {
		out[i] = dory.ScoredChunk{Chunk: results[i].chunk, Score: results[i].score}
	}
	return out, nil
}

// Delete removes chunks by their IDs.
func (s *SQLite) Delete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf(`DELETE FROM %s WHERE id IN (%s)`, s.tableName, strings.Join(placeholders, ","))
	_, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("store: delete: %w", err)
	}
	return nil
}

// marshalVector serializes a float32 vector as a JSON array string.
func marshalVector(vec []float32) (string, error) {
	if vec == nil {
		return "[]", nil
	}
	data, err := json.Marshal(vec)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// unmarshalVector deserializes a JSON array string into a float32 vector.
func unmarshalVector(s string) ([]float32, error) {
	var f64 []float64
	if err := json.Unmarshal([]byte(s), &f64); err != nil {
		return nil, err
	}
	f32 := make([]float32, len(f64))
	for i, v := range f64 {
		f32[i] = float32(v)
	}
	return f32, nil
}

// marshalMetadata serializes metadata as a JSON object string.
func marshalMetadata(meta map[string]any) (string, error) {
	if meta == nil {
		return "{}", nil
	}
	data, err := json.Marshal(meta)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// unmarshalMetadata deserializes a JSON object string into metadata.
func unmarshalMetadata(s string) (map[string]any, error) {
	var meta map[string]any
	if err := json.Unmarshal([]byte(s), &meta); err != nil {
		return nil, err
	}
	return meta, nil
}

// cosineSimilarity computes cosine similarity between two vectors.
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
