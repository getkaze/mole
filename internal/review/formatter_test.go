package review

import (
	"strings"
	"testing"

	"github.com/getkaze/mole/internal/llm"
	"github.com/getkaze/mole/internal/personality"
)

func TestFormat_BasicReview(t *testing.T) {
	resp := &llm.ReviewResponse{
		Summary: "Looks good overall.",
		Comments: []llm.InlineComment{
			{File: "main.go", Line: 10, Category: "Security", Subcategory: "SQL Injection", Severity: "critical", Message: "SQL injection"},
		},
	}

	engine := personality.New("mole", "en")
	result := Format(resp, engine, 85)

	if !strings.Contains(result.Body, "Mole Review") {
		t.Error("body should contain header")
	}
	if !strings.Contains(result.Body, "85") {
		t.Error("body should contain score")
	}
	if !strings.Contains(result.Body, "SQL injection") {
		t.Error("body should contain comment message")
	}
	if !strings.Contains(result.Body, "Security / SQL Injection") {
		t.Error("body should contain category / subcategory")
	}
	if len(result.Comments) != 1 {
		t.Errorf("comments = %d, want 1", len(result.Comments))
	}
}

func TestFormat_NoComments(t *testing.T) {
	resp := &llm.ReviewResponse{
		Summary:  "Clean code.",
		Comments: nil,
	}
	engine := personality.New("mole", "en")
	result := Format(resp, engine, 100)
	if !strings.Contains(result.Body, "Looking good") {
		t.Error("should show clean PR message for mole personality")
	}
}

func TestFormat_FormalPersonality(t *testing.T) {
	resp := &llm.ReviewResponse{
		Summary: "issues found",
		Comments: []llm.InlineComment{
			{File: "a.go", Line: 1, Category: "Bugs", Severity: "critical", Message: "null pointer"},
		},
	}
	engine := personality.New("formal", "en")
	result := Format(resp, engine, 85)

	if !strings.Contains(result.Body, "Quality Score") {
		t.Error("formal should show Quality Score badge")
	}
	if !strings.Contains(result.Body, "Critical") {
		t.Error("should contain severity label")
	}
}

func TestFormat_MinimalPersonality(t *testing.T) {
	resp := &llm.ReviewResponse{
		Summary: "issues",
		Comments: []llm.InlineComment{
			{File: "a.go", Line: 1, Category: "Bugs", Severity: "attention", Message: "potential issue"},
		},
	}
	engine := personality.New("minimal", "en")
	result := Format(resp, engine, 95)

	if !strings.Contains(result.Body, "95/100") {
		t.Error("minimal should show score")
	}
}

func TestFormat_SeverityBadges(t *testing.T) {
	resp := &llm.ReviewResponse{
		Summary: "issues",
		Comments: []llm.InlineComment{
			{File: "a.go", Line: 1, Category: "Bugs", Severity: "critical", Message: "null pointer"},
			{File: "b.go", Line: 2, Category: "Bugs", Severity: "attention", Message: "unhandled error"},
		},
	}
	engine := personality.New("mole", "en")
	result := Format(resp, engine, 80)

	if len(result.Comments) != 2 {
		t.Fatalf("got %d comments, want 2", len(result.Comments))
	}
	if !strings.Contains(result.Comments[0].Body, "Critical") {
		t.Error("critical should have Critical label")
	}
	if !strings.Contains(result.Comments[1].Body, "Attention") {
		t.Error("attention should have Attention label")
	}
}

func TestFormat_SubcategoryDisplay(t *testing.T) {
	resp := &llm.ReviewResponse{
		Summary: "issues",
		Comments: []llm.InlineComment{
			{File: "a.go", Line: 1, Category: "Security", Subcategory: "XSS", Severity: "critical", Message: "xss"},
			{File: "b.go", Line: 2, Category: "Bugs", Severity: "attention", Message: "unhandled error"},
		},
	}
	engine := personality.New("mole", "en")
	result := Format(resp, engine, 84)

	if !strings.Contains(result.Body, "Security / XSS") {
		t.Error("should show category / subcategory when subcategory present")
	}
	// No subcategory — should show just category
	if strings.Contains(result.Body, "Bugs /") {
		t.Error("should not show slash when no subcategory")
	}
}

func TestFormat_PortugueseBR(t *testing.T) {
	resp := &llm.ReviewResponse{
		Summary: "found issues",
		Comments: []llm.InlineComment{
			{File: "main.go", Line: 10, Category: "Security", Severity: "critical", Message: "SQL injection"},
		},
	}
	engine := personality.New("mole", "pt-BR")
	result := Format(resp, engine, 85)

	if !strings.Contains(result.Body, "toupeira") {
		t.Error("pt-BR mole should use Portuguese text")
	}
	if !strings.Contains(result.Comments[0].Body, "Crítico") {
		t.Error("inline comment should use Portuguese severity label")
	}
}

func TestFormat_Diagrams(t *testing.T) {
	resp := &llm.ReviewResponse{
		Summary:  "has diagrams",
		Comments: nil,
		Diagrams: []string{"sequenceDiagram\n    A->>B: call"},
	}
	engine := personality.New("mole", "en")
	result := Format(resp, engine, 100)

	if !strings.Contains(result.Body, "```mermaid") {
		t.Error("should contain mermaid code block")
	}
}
