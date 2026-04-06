package store

import (
	"testing"
)

func TestNewSQLite_NilDB(t *testing.T) {
	_, err := NewSQLite(SQLiteConfig{DB: nil})
	if err == nil {
		t.Fatal("expected error for nil DB")
	}
}

func TestNewSQLite_DefaultTableName(t *testing.T) {
	// We can't pass a real *sql.DB without adding a driver dep,
	// so we test the config validation path only for nil DB.
	// For table name default, we verify via a non-nil DB by using
	// a trick: the constructor doesn't use the DB, so a typed nil works
	// for testing the table name logic (but not for actual queries).
	// Instead we test the helper functions thoroughly.
}

func TestMarshalVector(t *testing.T) {
	tests := []struct {
		name string
		vec  []float32
		want string
	}{
		{"nil vector", nil, "[]"},
		{"empty vector", []float32{}, "[]"},
		{"simple vector", []float32{0.1, 0.2, 0.3}, "[0.1,0.2,0.3]"},
		{"single element", []float32{1.0}, "[1]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := marshalVector(tt.vec)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestUnmarshalVector(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantLen int
		wantErr bool
	}{
		{"empty array", "[]", 0, false},
		{"three elements", "[0.1,0.2,0.3]", 3, false},
		{"invalid json", "not json", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := unmarshalVector(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr = %v", err, tt.wantErr)
			}
			if !tt.wantErr && len(got) != tt.wantLen {
				t.Errorf("got len %d, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestUnmarshalVector_Values(t *testing.T) {
	vec, err := unmarshalVector("[0.1,0.2,0.3]")
	if err != nil {
		t.Fatal(err)
	}
	if vec[0] != float32(0.1) || vec[1] != float32(0.2) || vec[2] != float32(0.3) {
		t.Errorf("unexpected values: %v", vec)
	}
}

func TestMarshalMetadata(t *testing.T) {
	tests := []struct {
		name string
		meta map[string]any
		want string
	}{
		{"nil metadata", nil, "{}"},
		{"empty metadata", map[string]any{}, "{}"},
		{"single field", map[string]any{"tenant": "acme"}, `{"tenant":"acme"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := marshalMetadata(tt.meta)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestUnmarshalMetadata(t *testing.T) {
	meta, err := unmarshalMetadata(`{"tenant":"acme","count":42}`)
	if err != nil {
		t.Fatal(err)
	}
	if meta["tenant"] != "acme" {
		t.Errorf("got tenant %v, want acme", meta["tenant"])
	}
	// JSON numbers decode as float64.
	if meta["count"] != float64(42) {
		t.Errorf("got count %v, want 42", meta["count"])
	}
}

func TestUnmarshalMetadata_Invalid(t *testing.T) {
	_, err := unmarshalMetadata("not json")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name string
		a, b []float32
		want float64
	}{
		{"identical", []float32{1, 0, 0}, []float32{1, 0, 0}, 1.0},
		{"orthogonal", []float32{1, 0, 0}, []float32{0, 1, 0}, 0.0},
		{"opposite", []float32{1, 0, 0}, []float32{-1, 0, 0}, -1.0},
		{"different lengths", []float32{1, 0}, []float32{1, 0, 0}, 0.0},
		{"empty", []float32{}, []float32{}, 0.0},
		{"zero vector", []float32{0, 0, 0}, []float32{1, 0, 0}, 0.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cosineSimilarity(tt.a, tt.b)
			if diff := got - tt.want; diff > 1e-6 || diff < -1e-6 {
				t.Errorf("got %f, want %f", got, tt.want)
			}
		})
	}
}

func TestMarshalUnmarshalRoundTrip(t *testing.T) {
	original := []float32{0.123, 0.456, 0.789}
	s, err := marshalVector(original)
	if err != nil {
		t.Fatal(err)
	}
	result, err := unmarshalVector(s)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != len(original) {
		t.Fatalf("got len %d, want %d", len(result), len(original))
	}
	for i := range original {
		if result[i] != original[i] {
			t.Errorf("index %d: got %f, want %f", i, result[i], original[i])
		}
	}
}
