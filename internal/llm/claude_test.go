package llm

import "testing"

func TestMaxTokensForModel(t *testing.T) {
	tests := []struct {
		model string
		want  int64
	}{
		{"claude-opus-4-20250514", 32000},
		{"claude-sonnet-4-20250514", 16000},
		{"anything-else", 16000},
		{"", 16000},
	}
	for _, tt := range tests {
		if got := maxTokensForModel(tt.model); got != tt.want {
			t.Errorf("maxTokensForModel(%q) = %d, want %d", tt.model, got, tt.want)
		}
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		s, substr string
		want      bool
	}{
		{"hello world", "world", true},
		{"hello", "hello", true},
		{"hello", "xyz", false},
		{"", "a", false},
		{"a", "", true},
		{"", "", true},
	}
	for _, tt := range tests {
		if got := contains(tt.s, tt.substr); got != tt.want {
			t.Errorf("contains(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
		}
	}
}
