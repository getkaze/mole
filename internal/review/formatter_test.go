package review

import (
	"strings"
	"testing"

	"github.com/getkaze/kite/internal/llm"
)

func TestFormat_BasicReview(t *testing.T) {
	resp := &llm.ReviewResponse{
		Summary: "Looks good overall.",
		Comments: []llm.InlineComment{
			{File: "main.go", Line: 10, Category: "security", Severity: "must-fix", Message: "SQL injection"},
		},
		Suggestions: []string{"Add tests"},
	}

	result := Format(resp, "", "en")

	if !strings.Contains(result.Body, "Kite Review") {
		t.Error("body should contain header")
	}
	if !strings.Contains(result.Body, "Looks good overall.") {
		t.Error("body should contain summary")
	}
	if !strings.Contains(result.Body, "SQL injection") {
		t.Error("body should contain comment")
	}
	if !strings.Contains(result.Body, "Add tests") {
		t.Error("body should contain suggestion")
	}
	if len(result.Comments) != 1 {
		t.Errorf("comments = %d, want 1", len(result.Comments))
	}
	if !strings.Contains(result.Comments[0].Body, "`security`") {
		t.Error("inline comment should contain category in code format")
	}
	if !strings.Contains(result.Comments[0].Body, "---\n") {
		t.Error("inline comment should contain separator")
	}
}

func TestFormat_NoComments(t *testing.T) {
	resp := &llm.ReviewResponse{
		Summary:  "Clean code.",
		Comments: nil,
	}
	result := Format(resp, "", "en")
	if !strings.Contains(result.Body, "No issues found") {
		t.Error("should show 'No issues found' when no comments")
	}
}

func TestFormat_WithSplitNote(t *testing.T) {
	resp := &llm.ReviewResponse{Summary: "ok"}
	result := Format(resp, "Reviewed in 3 groups due to PR size.", "en")
	if !strings.Contains(result.Body, "3 groups") {
		t.Error("body should contain split note")
	}
}

func TestFormat_SeverityBadges(t *testing.T) {
	resp := &llm.ReviewResponse{
		Summary: "issues",
		Comments: []llm.InlineComment{
			{File: "a.go", Line: 1, Category: "bug", Severity: "must-fix", Message: "null pointer"},
			{File: "b.go", Line: 2, Category: "style", Severity: "should-fix", Message: "naming"},
			{File: "c.go", Line: 3, Category: "style", Severity: "consider", Message: "nit"},
			{File: "d.go", Line: 4, Category: "style", Severity: "unknown", Message: "fallback"},
		},
	}
	result := Format(resp, "", "en")

	if len(result.Comments) != 4 {
		t.Fatalf("got %d comments, want 4", len(result.Comments))
	}
	if !strings.Contains(result.Comments[0].Body, "Must Fix") {
		t.Error("must-fix should have Must Fix badge")
	}
	if !strings.Contains(result.Comments[1].Body, "Should Fix") {
		t.Error("should-fix should have Should Fix badge")
	}
	if !strings.Contains(result.Comments[2].Body, "Consider") {
		t.Error("consider should have Consider badge")
	}
	// Unknown severity falls back to consider
	if !strings.Contains(result.Comments[3].Body, "Consider") {
		t.Error("unknown severity should fall back to Consider badge")
	}
}

func TestFormat_ListLayout(t *testing.T) {
	resp := &llm.ReviewResponse{
		Summary: "found issues",
		Comments: []llm.InlineComment{
			{File: "main.go", Line: 10, Category: "security", Severity: "must-fix", Message: "SQL injection"},
		},
	}
	result := Format(resp, "", "en")

	// Should use list format: "**1.** `file:line` — badge **label** · category"
	if !strings.Contains(result.Body, "**1.** `main.go:10`") {
		t.Error("body should use list format with file:line")
	}
	if !strings.Contains(result.Body, "Must Fix") {
		t.Error("body should contain severity label")
	}
	// Should NOT contain table separators
	if strings.Contains(result.Body, "|---|") {
		t.Error("body should not contain table format")
	}
}

func TestFormat_PortugueseBR(t *testing.T) {
	resp := &llm.ReviewResponse{
		Summary: "found issues",
		Comments: []llm.InlineComment{
			{File: "main.go", Line: 10, Category: "security", Severity: "must-fix", Message: "SQL injection"},
		},
	}
	result := Format(resp, "", "pt-BR")

	if !strings.Contains(result.Body, "Resumo") {
		t.Error("body should contain Portuguese summary header")
	}
	if !strings.Contains(result.Body, "Problemas Encontrados") {
		t.Error("body should contain Portuguese issues header")
	}
	if !strings.Contains(result.Comments[0].Body, "Corrigir") {
		t.Error("inline comment should use Portuguese severity label")
	}
}

func TestFormat_UnknownLangFallsBackToEnglish(t *testing.T) {
	resp := &llm.ReviewResponse{
		Summary:  "Clean.",
		Comments: nil,
	}
	result := Format(resp, "", "fr-FR")
	if !strings.Contains(result.Body, "No issues found") {
		t.Error("unknown lang should fall back to English")
	}
}
