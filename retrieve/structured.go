package retrieve

import (
	"context"
	"fmt"

	"github.com/i33ym/dory"
)

// StructuredConfig holds configuration for a Structured retriever.
type StructuredConfig struct {
	// TextToSQL converts a natural language question to a SQL query.
	TextToSQL func(ctx context.Context, question string) (string, error)

	// ExecSQL executes a SQL query and returns the result rows.
	ExecSQL func(ctx context.Context, sql string) ([]map[string]any, error)

	// SourceDocID is the source document ID assigned to all returned rows.
	SourceDocID string
}

// Structured retriever converts natural language queries to SQL,
// executes the SQL, and returns the result rows as StructuredRow units.
type Structured struct {
	config StructuredConfig
}

// NewStructured creates a new structured (text-to-SQL) retriever.
func NewStructured(config StructuredConfig) *Structured {
	return &Structured{config: config}
}

// Retrieve implements [dory.Retriever].
func (s *Structured) Retrieve(ctx context.Context, q dory.Query) ([]dory.RetrievedUnit, error) {
	sql, err := s.config.TextToSQL(ctx, q.Text)
	if err != nil {
		return nil, fmt.Errorf("text-to-sql: %w", err)
	}

	rows, err := s.config.ExecSQL(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("exec-sql: %w", err)
	}

	units := make([]dory.RetrievedUnit, len(rows))
	for i, row := range rows {
		id := fmt.Sprintf("row-%d", i)
		sr := dory.NewStructuredRow(id, s.config.SourceDocID, row, nil)
		units[i] = sr.WithScore("structured", 1.0)
	}
	return units, nil
}
