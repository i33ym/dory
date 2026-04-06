package filter

import (
	"testing"

	"github.com/i33ym/dory"
)

func TestMatchEq(t *testing.T) {
	meta := map[string]any{"tenant": "acme", "count": 42}

	if !Match(meta, &dory.MetadataFilter{Field: "tenant", Op: dory.FilterOpEq, Value: "acme"}) {
		t.Error("expected eq match for tenant=acme")
	}
	if Match(meta, &dory.MetadataFilter{Field: "tenant", Op: dory.FilterOpEq, Value: "other"}) {
		t.Error("expected no match for tenant=other")
	}
	if !Match(meta, &dory.MetadataFilter{Field: "count", Op: dory.FilterOpEq, Value: 42}) {
		t.Error("expected eq match for count=42")
	}
}

func TestMatchIn(t *testing.T) {
	meta := map[string]any{"status": "active"}

	if !Match(meta, &dory.MetadataFilter{
		Field: "status", Op: dory.FilterOpIn, Value: []string{"active", "pending"},
	}) {
		t.Error("expected in match for status in [active, pending]")
	}
	if Match(meta, &dory.MetadataFilter{
		Field: "status", Op: dory.FilterOpIn, Value: []string{"archived"},
	}) {
		t.Error("expected no match for status in [archived]")
	}
}

func TestMatchAnyOf(t *testing.T) {
	meta := map[string]any{"tags": []any{"go", "rust", "python"}}

	if !Match(meta, &dory.MetadataFilter{
		Field: "tags", Op: dory.FilterOpAnyOf, Value: []string{"go", "java"},
	}) {
		t.Error("expected any_of match when tags contains go")
	}
	if Match(meta, &dory.MetadataFilter{
		Field: "tags", Op: dory.FilterOpAnyOf, Value: []string{"java", "c++"},
	}) {
		t.Error("expected no any_of match when no overlap")
	}
}

func TestMatchAnyOfStringSlice(t *testing.T) {
	meta := map[string]any{"tags": []string{"go", "rust"}}

	if !Match(meta, &dory.MetadataFilter{
		Field: "tags", Op: dory.FilterOpAnyOf, Value: []string{"rust"},
	}) {
		t.Error("expected any_of match with []string field")
	}
}

func TestMatchMissingField(t *testing.T) {
	meta := map[string]any{"tenant": "acme"}
	if Match(meta, &dory.MetadataFilter{Field: "missing", Op: dory.FilterOpEq, Value: "x"}) {
		t.Error("expected no match for missing field")
	}
}

func TestMatchNilMetadata(t *testing.T) {
	if Match(nil, &dory.MetadataFilter{Field: "f", Op: dory.FilterOpEq, Value: "v"}) {
		t.Error("expected no match for nil metadata")
	}
}

func TestMatchNilFilter(t *testing.T) {
	meta := map[string]any{"f": "v"}
	if Match(meta, nil) {
		t.Error("expected no match for nil filter")
	}
}

func TestMatchAll(t *testing.T) {
	meta := map[string]any{"tenant": "acme", "status": "active"}

	filters := []dory.MetadataFilter{
		{Field: "tenant", Op: dory.FilterOpEq, Value: "acme"},
		{Field: "status", Op: dory.FilterOpEq, Value: "active"},
	}
	if !MatchAll(meta, filters) {
		t.Error("expected MatchAll to pass when all filters match")
	}

	filters = append(filters, dory.MetadataFilter{
		Field: "missing", Op: dory.FilterOpEq, Value: "x",
	})
	if MatchAll(meta, filters) {
		t.Error("expected MatchAll to fail when one filter doesn't match")
	}
}

func TestMatchAllEmpty(t *testing.T) {
	meta := map[string]any{"f": "v"}
	if !MatchAll(meta, nil) {
		t.Error("expected MatchAll with no filters to return true")
	}
	if !MatchAll(meta, []dory.MetadataFilter{}) {
		t.Error("expected MatchAll with empty filters to return true")
	}
}
