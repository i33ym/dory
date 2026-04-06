package embed

import (
	"context"
	"os"
	"testing"
)

func skipIfNoKey(t *testing.T) {
	t.Helper()
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("OPENAI_API_KEY not set, skipping integration test")
	}
}

func TestOpenAI_Embed(t *testing.T) {
	skipIfNoKey(t)
	ctx := context.Background()
	e := NewOpenAI("text-embedding-3-small")

	vec, err := e.Embed(ctx, "hello world")
	if err != nil {
		t.Fatal(err)
	}
	if len(vec) != e.Dimensions() {
		t.Fatalf("got %d dimensions, want %d", len(vec), e.Dimensions())
	}

	// Vectors should be non-zero.
	allZero := true
	for _, v := range vec {
		if v != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("embedding is all zeros")
	}
}

func TestOpenAI_EmbedBatch(t *testing.T) {
	skipIfNoKey(t)
	ctx := context.Background()
	e := NewOpenAI("text-embedding-3-small")

	texts := []string{"the cat sat on the mat", "the dog chased the ball"}
	vecs, err := e.EmbedBatch(ctx, texts)
	if err != nil {
		t.Fatal(err)
	}
	if len(vecs) != 2 {
		t.Fatalf("got %d vectors, want 2", len(vecs))
	}
	for i, vec := range vecs {
		if len(vec) != e.Dimensions() {
			t.Errorf("vector %d: got %d dimensions, want %d", i, len(vec), e.Dimensions())
		}
	}
}

func TestOpenAI_EmbedBatch_Empty(t *testing.T) {
	skipIfNoKey(t)
	e := NewOpenAI("text-embedding-3-small")

	vecs, err := e.EmbedBatch(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if vecs != nil {
		t.Errorf("expected nil for empty input, got %v", vecs)
	}
}

func TestOpenAI_SimilarTextsCluster(t *testing.T) {
	skipIfNoKey(t)
	ctx := context.Background()
	e := NewOpenAI("text-embedding-3-small")

	vecs, err := e.EmbedBatch(ctx, []string{
		"the weather is sunny today",
		"it is a bright and clear day",
		"quantum mechanics describes subatomic particles",
	})
	if err != nil {
		t.Fatal(err)
	}

	simAB := cosine(vecs[0], vecs[1])
	simAC := cosine(vecs[0], vecs[2])

	if simAB <= simAC {
		t.Errorf("expected similar texts to be closer: sim(A,B)=%f <= sim(A,C)=%f", simAB, simAC)
	}
}

func TestOpenAI_Dimensions(t *testing.T) {
	tests := []struct {
		model string
		want  int
	}{
		{"text-embedding-3-small", 1536},
		{"text-embedding-3-large", 3072},
		{"text-embedding-ada-002", 1536},
	}
	for _, tt := range tests {
		e := NewOpenAI(tt.model)
		if e.Dimensions() != tt.want {
			t.Errorf("%s: got %d, want %d", tt.model, e.Dimensions(), tt.want)
		}
	}
}

func TestOpenAI_NoAPIKey(t *testing.T) {
	orig := os.Getenv("OPENAI_API_KEY")
	os.Unsetenv("OPENAI_API_KEY")
	defer os.Setenv("OPENAI_API_KEY", orig)

	e := NewOpenAI("text-embedding-3-small")
	_, err := e.Embed(context.Background(), "hello")
	if err == nil {
		t.Fatal("expected error when API key is not set")
	}
}

// cosine computes cosine similarity for test assertions.
func cosine(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (sqrt(na) * sqrt(nb))
}

func sqrt(x float64) float64 {
	z := x / 2
	for range 100 {
		z = z - (z*z-x)/(2*z)
	}
	return z
}
