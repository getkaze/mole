package review

import (
	"strings"
	"testing"
)

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"", 0},
		{"abcd", 1},
		{"abcdefgh", 2},
		{strings.Repeat("x", 400), 100},
	}
	for _, tt := range tests {
		if got := EstimateTokens(tt.input); got != tt.want {
			t.Errorf("EstimateTokens(%d chars) = %d, want %d", len(tt.input), got, tt.want)
		}
	}
}
