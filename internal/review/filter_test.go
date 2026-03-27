package review

import (
	"testing"

	"github.com/getkaze/mole/internal/llm"
)

func TestFilterComments_MinSeverity(t *testing.T) {
	comments := []llm.InlineComment{
		{File: "a.go", Severity: "suggestion", Message: "nit"},
		{File: "b.go", Severity: "attention", Message: "warn"},
		{File: "c.go", Severity: "critical", Message: "bad"},
		{File: "d.go", Severity: "suggestion", Message: "nit2"},
		{File: "e.go", Severity: "suggestion", Message: "nit3"},
	}

	result := FilterComments(comments, "attention", nil, 0)
	if len(result) != 2 {
		t.Errorf("got %d comments, want 2 (attention + critical)", len(result))
	}
}

func TestFilterComments_MinSeverityCritical(t *testing.T) {
	comments := []llm.InlineComment{
		{File: "a.go", Severity: "suggestion"},
		{File: "b.go", Severity: "attention"},
		{File: "c.go", Severity: "critical"},
	}

	result := FilterComments(comments, "critical", nil, 0)
	if len(result) != 1 {
		t.Errorf("got %d, want 1 (critical only)", len(result))
	}
}

func TestFilterComments_NoFilter(t *testing.T) {
	comments := []llm.InlineComment{
		{File: "a.go", Severity: "suggestion"},
		{File: "b.go", Severity: "attention"},
	}
	result := FilterComments(comments, "", nil, 0)
	if len(result) != 2 {
		t.Errorf("got %d, want 2 (no filter)", len(result))
	}
}

func TestFilterComments_MaxComments(t *testing.T) {
	comments := []llm.InlineComment{
		{File: "a.go", Severity: "suggestion", Message: "1"},
		{File: "b.go", Severity: "critical", Message: "2"},
		{File: "c.go", Severity: "attention", Message: "3"},
		{File: "d.go", Severity: "critical", Message: "4"},
		{File: "e.go", Severity: "suggestion", Message: "5"},
	}

	result := FilterComments(comments, "", nil, 3)
	if len(result) != 3 {
		t.Fatalf("got %d, want 3", len(result))
	}
	// Should keep highest severity first
	if result[0].Severity != "critical" {
		t.Errorf("first comment should be critical, got %q", result[0].Severity)
	}
	if result[1].Severity != "critical" {
		t.Errorf("second comment should be critical, got %q", result[1].Severity)
	}
}

func TestFilterComments_IgnorePatterns(t *testing.T) {
	comments := []llm.InlineComment{
		{File: "internal/foo_test.go", Severity: "critical"},
		{File: "internal/foo.go", Severity: "critical"},
		{File: "vendor/lib.go", Severity: "attention"},
	}

	result := FilterComments(comments, "", []string{"*_test.go", "vendor/*"}, 0)
	if len(result) != 1 {
		t.Errorf("got %d, want 1 (only foo.go)", len(result))
	}
	if result[0].File != "internal/foo.go" {
		t.Errorf("expected foo.go, got %q", result[0].File)
	}
}

func TestFilterComments_CombinedFilters(t *testing.T) {
	comments := []llm.InlineComment{
		{File: "a.go", Severity: "suggestion"},
		{File: "b_test.go", Severity: "critical"},
		{File: "c.go", Severity: "critical"},
		{File: "d.go", Severity: "attention"},
		{File: "e.go", Severity: "attention"},
	}

	// min_severity=attention, ignore test files, max 2
	result := FilterComments(comments, "attention", []string{"*_test.go"}, 2)
	if len(result) != 2 {
		t.Fatalf("got %d, want 2", len(result))
	}
	// critical first, then attention
	if result[0].Severity != "critical" {
		t.Errorf("first should be critical, got %q", result[0].Severity)
	}
}
