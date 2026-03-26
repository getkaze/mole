package review

import (
	"strings"
	"testing"

	"github.com/getkaze/kite/internal/llm"
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

func TestSplitDiffs_SingleGroup(t *testing.T) {
	diffs := []llm.FileDiff{
		{Filename: "a.go", Patch: strings.Repeat("x", 400)},
		{Filename: "b.go", Patch: strings.Repeat("x", 400)},
	}
	groups := SplitDiffs(diffs, 0)
	if len(groups) != 1 {
		t.Fatalf("got %d groups, want 1", len(groups))
	}
	if len(groups[0].Files) != 2 {
		t.Errorf("group has %d files, want 2", len(groups[0].Files))
	}
}

func TestSplitDiffs_MultipleGroups(t *testing.T) {
	// Each file is ~37500 tokens. With contextTokens=0, available=149500.
	// 4 files = 150000 tokens, which exceeds available.
	bigPatch := strings.Repeat("x", 150000) // 37500 tokens
	diffs := []llm.FileDiff{
		{Filename: "a.go", Patch: bigPatch},
		{Filename: "b.go", Patch: bigPatch},
		{Filename: "c.go", Patch: bigPatch},
		{Filename: "d.go", Patch: bigPatch},
	}
	groups := SplitDiffs(diffs, 0)
	if len(groups) < 2 {
		t.Errorf("got %d groups, want >= 2 for large diffs", len(groups))
	}
}

func TestSplitDiffs_EmptyPatch(t *testing.T) {
	diffs := []llm.FileDiff{
		{Filename: "a.go", Patch: ""},
	}
	groups := SplitDiffs(diffs, 0)
	if len(groups) != 1 {
		t.Fatalf("got %d groups, want 1", len(groups))
	}
	// Empty patch gets minimum 100 tokens
	if groups[0].TotalTokens != 100 {
		t.Errorf("total tokens = %d, want 100", groups[0].TotalTokens)
	}
}

func TestSplitDiffs_Empty(t *testing.T) {
	groups := SplitDiffs(nil, 0)
	if len(groups) != 0 {
		t.Errorf("got %d groups, want 0", len(groups))
	}
}

func TestAggregateResponses_Single(t *testing.T) {
	resp := &llm.ReviewResponse{Summary: "ok"}
	result := AggregateResponses([]*llm.ReviewResponse{resp})
	if result != resp {
		t.Error("single response should return same pointer")
	}
}

func TestAggregateResponses_Multiple(t *testing.T) {
	r1 := &llm.ReviewResponse{
		Summary:     "first",
		Comments:    []llm.InlineComment{{File: "a.go", Line: 1}},
		Suggestions: []string{"use gofmt"},
		Usage:       llm.TokenUsage{InputTokens: 100, OutputTokens: 50},
	}
	r2 := &llm.ReviewResponse{
		Summary:     "second",
		Comments:    []llm.InlineComment{{File: "b.go", Line: 2}},
		Suggestions: []string{"use gofmt", "add tests"},
		Usage:       llm.TokenUsage{InputTokens: 200, OutputTokens: 100},
	}
	result := AggregateResponses([]*llm.ReviewResponse{r1, r2})

	if !strings.Contains(result.Summary, "first") || !strings.Contains(result.Summary, "second") {
		t.Errorf("summary should contain both, got %q", result.Summary)
	}
	if len(result.Comments) != 2 {
		t.Errorf("comments = %d, want 2", len(result.Comments))
	}
	// Deduplicated suggestions
	if len(result.Suggestions) != 2 {
		t.Errorf("suggestions = %d, want 2 (deduplicated)", len(result.Suggestions))
	}
	if result.Usage.InputTokens != 300 {
		t.Errorf("input tokens = %d, want 300", result.Usage.InputTokens)
	}
	if result.Usage.OutputTokens != 150 {
		t.Errorf("output tokens = %d, want 150", result.Usage.OutputTokens)
	}
}

func TestSplitNote(t *testing.T) {
	if got := SplitNote(1, "en"); got != "" {
		t.Errorf("SplitNote(1) = %q, want empty", got)
	}
	if got := SplitNote(3, "en"); got == "" {
		t.Error("SplitNote(3) should not be empty")
	}
	if got := SplitNote(3, "pt-BR"); !strings.Contains(got, "grupos") {
		t.Errorf("SplitNote pt-BR should contain 'grupos', got %q", got)
	}
}
