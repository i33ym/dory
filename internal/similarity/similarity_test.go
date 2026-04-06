package similarity

import (
	"math"
	"math/rand"
	"testing"
)

func TestCosine(t *testing.T) {
	t.Run("identical vectors return 1", func(t *testing.T) {
		v := []float32{1, 2, 3}
		got := Cosine(v, v)
		if math.Abs(got-1.0) > 1e-9 {
			t.Errorf("got %f, want 1.0", got)
		}
	})

	t.Run("opposite vectors return -1", func(t *testing.T) {
		a := []float32{1, 0, 0}
		b := []float32{-1, 0, 0}
		got := Cosine(a, b)
		if math.Abs(got-(-1.0)) > 1e-9 {
			t.Errorf("got %f, want -1.0", got)
		}
	})

	t.Run("orthogonal vectors return 0", func(t *testing.T) {
		a := []float32{1, 0, 0}
		b := []float32{0, 1, 0}
		got := Cosine(a, b)
		if math.Abs(got) > 1e-9 {
			t.Errorf("got %f, want 0.0", got)
		}
	})

	t.Run("different lengths return 0", func(t *testing.T) {
		a := []float32{1, 2}
		b := []float32{1, 2, 3}
		got := Cosine(a, b)
		if got != 0 {
			t.Errorf("got %f, want 0", got)
		}
	})

	t.Run("empty vectors return 0", func(t *testing.T) {
		got := Cosine([]float32{}, []float32{})
		if got != 0 {
			t.Errorf("got %f, want 0", got)
		}
	})

	t.Run("zero vector returns 0", func(t *testing.T) {
		a := []float32{0, 0, 0}
		b := []float32{1, 2, 3}
		got := Cosine(a, b)
		if got != 0 {
			t.Errorf("got %f, want 0", got)
		}
	})

	t.Run("known angle", func(t *testing.T) {
		// 45 degree angle: cos(45°) ≈ 0.7071
		a := []float32{1, 0}
		b := []float32{1, 1}
		got := Cosine(a, b)
		want := 1.0 / math.Sqrt(2)
		if math.Abs(got-want) > 1e-6 {
			t.Errorf("got %f, want %f", got, want)
		}
	})

	t.Run("magnitude invariant", func(t *testing.T) {
		a := []float32{1, 2, 3}
		b := []float32{4, 5, 6}
		sim1 := Cosine(a, b)

		// Scale a by 100 — cosine should be the same.
		scaled := []float32{100, 200, 300}
		sim2 := Cosine(scaled, b)
		if math.Abs(sim1-sim2) > 1e-6 {
			t.Errorf("cosine changed with magnitude: %f vs %f", sim1, sim2)
		}
	})
}

func BenchmarkCosine(b *testing.B) {
	const dims = 1536
	rng := rand.New(rand.NewSource(42))

	a := make([]float32, dims)
	v := make([]float32, dims)
	for i := range a {
		a[i] = rng.Float32()
		v[i] = rng.Float32()
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Cosine(a, v)
	}
}
