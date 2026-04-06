package tokenizer

import (
	"testing"
)

func TestTokenize(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "simple text",
			input: "hello world",
			want:  []string{"hello", "world"},
		},
		{
			name:  "with punctuation",
			input: "Hello, world!",
			want:  []string{"hello", ",", "world", "!"},
		},
		{
			name:  "multiple spaces",
			input: "hello   world   foo",
			want:  []string{"hello", "world", "foo"},
		},
		{
			name:  "empty string",
			input: "",
			want:  nil,
		},
		{
			name:  "uppercase lowered",
			input: "Hello World",
			want:  []string{"hello", "world"},
		},
		{
			name:  "punctuation only",
			input: "...",
			want:  []string{".", ".", "."},
		},
		{
			name:  "mixed punctuation and words",
			input: "it's a test-case (really)",
			want:  []string{"it", "'", "s", "a", "test", "-", "case", "(", "really", ")"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Tokenize(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("Tokenize(%q) = %v (len %d), want %v (len %d)",
					tt.input, got, len(got), tt.want, len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("Tokenize(%q)[%d] = %q, want %q",
						tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestCount(t *testing.T) {
	tests := []struct {
		input string
	}{
		{"hello world"},
		{"Hello, world!"},
		{""},
		{"one"},
	}

	for _, tt := range tests {
		got := Count(tt.input)
		want := len(Tokenize(tt.input))
		if got != want {
			t.Errorf("Count(%q) = %d, want %d", tt.input, got, want)
		}
	}
}

func TestCountApprox(t *testing.T) {
	tests := []struct {
		input   string
		wantMin int
		wantMax int
	}{
		{"", 0, 0},
		{"hello world", 2, 4},               // 11 chars / 4 = 2
		{"this is a longer sentence", 5, 8}, // 25 chars / 4 = 6
	}

	for _, tt := range tests {
		got := CountApprox(tt.input)
		if got < tt.wantMin || got > tt.wantMax {
			t.Errorf("CountApprox(%q) = %d, want in [%d, %d]",
				tt.input, got, tt.wantMin, tt.wantMax)
		}
	}
}
