package review

import (
	"testing"

	"github.com/getkaze/mole/internal/llm"
)

func TestParseHunkHeader(t *testing.T) {
	tests := []struct {
		line      string
		wantStart int
		wantCount int
	}{
		{"@@ -10,5 +20,8 @@ func main()", 20, 8},
		{"@@ -1 +1 @@", 1, 1},
		{"@@ -0,0 +1,3 @@", 1, 3},
		{"invalid", 0, 0},
	}
	for _, tt := range tests {
		start, count := parseHunkHeader(tt.line)
		if start != tt.wantStart || count != tt.wantCount {
			t.Errorf("parseHunkHeader(%q) = (%d, %d), want (%d, %d)",
				tt.line, start, count, tt.wantStart, tt.wantCount)
		}
	}
}

func TestValidateComments_AdditionLineAccepted(t *testing.T) {
	// Line 1 is addition (+), line 2 is context ( ), line 3 is addition (+)
	diffs := []llm.FileDiff{
		{Filename: "main.go", Patch: "@@ -0,0 +1,3 @@\n+package main\n \n+func main() {}"},
	}
	comments := []llm.InlineComment{
		{File: "main.go", Line: 1, Message: "on addition line"},
	}
	valid := ValidateComments(comments, diffs)
	if len(valid) != 1 {
		t.Errorf("got %d valid comments, want 1", len(valid))
	}
}

func TestValidateComments_ContextLineRejected(t *testing.T) {
	// Line 2 is a context line (space prefix) — must be rejected
	diffs := []llm.FileDiff{
		{Filename: "main.go", Patch: "@@ -1,3 +1,3 @@\n+new line\n unchanged\n+another new"},
	}
	comments := []llm.InlineComment{
		{File: "main.go", Line: 2, Message: "on context line"},
	}
	valid := ValidateComments(comments, diffs)
	if len(valid) != 0 {
		t.Errorf("got %d valid comments, want 0 (context line should be rejected)", len(valid))
	}
}

func TestValidateComments_DeletionDoesNotAdvanceLine(t *testing.T) {
	// Hunk: +1,2 means new file lines 1-2
	// +added (line 1), -deleted (no new line), +added (line 2)
	diffs := []llm.FileDiff{
		{Filename: "main.go", Patch: "@@ -1,3 +1,2 @@\n+first\n-removed\n+second"},
	}

	// Line 1 = addition, Line 2 = addition (deletion didn't advance)
	comments := []llm.InlineComment{
		{File: "main.go", Line: 1, Message: "first addition"},
		{File: "main.go", Line: 2, Message: "second addition"},
	}
	valid := ValidateComments(comments, diffs)
	if len(valid) != 2 {
		t.Errorf("got %d valid comments, want 2", len(valid))
	}
}

func TestValidateComments_LineOutsideHunkRejected(t *testing.T) {
	diffs := []llm.FileDiff{
		{Filename: "main.go", Patch: "@@ -1,3 +1,5 @@\n+func main() {}"},
	}
	comments := []llm.InlineComment{
		{File: "main.go", Line: 100, Message: "way outside"},
	}
	valid := ValidateComments(comments, diffs)
	if len(valid) != 0 {
		t.Errorf("got %d valid comments, want 0", len(valid))
	}
}

func TestValidateComments_FileNotInDiff(t *testing.T) {
	diffs := []llm.FileDiff{
		{Filename: "main.go", Patch: "@@ -1,3 +1,5 @@\n+code"},
	}
	comments := []llm.InlineComment{
		{File: "other.go", Line: 1, Message: "wrong file"},
	}
	valid := ValidateComments(comments, diffs)
	if len(valid) != 0 {
		t.Errorf("got %d valid comments, want 0", len(valid))
	}
}

func TestValidateComments_MultipleHunks(t *testing.T) {
	patch := "@@ -1,3 +1,4 @@\n+first addition\n context\n context\n+second addition\n@@ -20,2 +21,3 @@\n context\n+third addition\n context"
	diffs := []llm.FileDiff{
		{Filename: "main.go", Patch: patch},
	}
	comments := []llm.InlineComment{
		{File: "main.go", Line: 1, Message: "first hunk addition"},   // +
		{File: "main.go", Line: 2, Message: "first hunk context"},    // context — reject
		{File: "main.go", Line: 4, Message: "first hunk addition 2"}, // +
		{File: "main.go", Line: 22, Message: "second hunk addition"}, // +
		{File: "main.go", Line: 21, Message: "second hunk context"},  // context — reject
	}
	valid := ValidateComments(comments, diffs)
	if len(valid) != 3 {
		t.Errorf("got %d valid comments, want 3 (only addition lines)", len(valid))
	}
}

func TestBuildAdditionLines_EmptyPatch(t *testing.T) {
	diffs := []llm.FileDiff{
		{Filename: "empty.go", Patch: ""},
	}
	result := buildAdditionLines(diffs)
	if lines, ok := result["empty.go"]; ok && len(lines) > 0 {
		t.Error("empty patch should have no addition lines")
	}
}
