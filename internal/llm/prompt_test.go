package llm

import (
	"strings"
	"testing"
)

func TestBuildPrompt_Standard(t *testing.T) {
	diffs := []FileDiff{
		{Filename: "main.go", Status: "modified", Patch: "+new line"},
	}
	system, user := BuildPrompt(diffs, "", "", "", "", false)

	if !strings.Contains(system, "Mole") {
		t.Error("system prompt should mention Mole")
	}
	if strings.Contains(system, "deep review") {
		t.Error("standard prompt should not mention deep review")
	}
	if !strings.Contains(user, "main.go") {
		t.Error("user prompt should contain filename")
	}
	if !strings.Contains(user, "+new line") {
		t.Error("user prompt should contain patch content")
	}
}

func TestBuildPrompt_Deep(t *testing.T) {
	diffs := []FileDiff{
		{Filename: "main.go", Status: "added", Patch: "+code"},
	}
	system, _ := BuildPrompt(diffs, "", "", "", "", true)

	if !strings.Contains(system, "deep review") {
		t.Error("deep prompt should mention deep review")
	}
	if !strings.Contains(system, "diagrams") {
		t.Error("deep prompt should mention diagrams")
	}
}

func TestBuildPrompt_WithContext(t *testing.T) {
	diffs := []FileDiff{
		{Filename: "a.go", Status: "modified", Patch: "+x"},
	}
	_, user := BuildPrompt(diffs, "project rules here", "", "", "", false)

	if !strings.Contains(user, "Project Context") {
		t.Error("should include project context header")
	}
	if !strings.Contains(user, "project rules here") {
		t.Error("should include project context content")
	}
}

func TestBuildPrompt_WithInstructions(t *testing.T) {
	diffs := []FileDiff{
		{Filename: "a.go", Status: "modified", Patch: "+x"},
	}
	_, user := BuildPrompt(diffs, "", "Always check for rate limiting", "", "", false)

	if !strings.Contains(user, "Repository-Specific Instructions") {
		t.Error("should include instructions header")
	}
	if !strings.Contains(user, "Always check for rate limiting") {
		t.Error("should include instruction content")
	}
}

func TestBuildPrompt_NoInstructions(t *testing.T) {
	diffs := []FileDiff{
		{Filename: "a.go", Status: "modified", Patch: "+x"},
	}
	_, user := BuildPrompt(diffs, "", "", "", "", false)

	if strings.Contains(user, "Repository-Specific Instructions") {
		t.Error("should not include instructions section when empty")
	}
}

func TestBuildPrompt_TooLargeFile(t *testing.T) {
	diffs := []FileDiff{
		{Filename: "big.go", Status: "modified", Patch: "", TooLarge: true},
	}
	_, user := BuildPrompt(diffs, "", "", "", "", false)

	if !strings.Contains(user, "too large to display") {
		t.Error("should note too-large file")
	}
}

func TestNumberDiffLines(t *testing.T) {
	patch := "@@ -10,5 +20,8 @@\n+added line\n context line\n-deleted line\n+another added"
	got := numberDiffLines(patch)
	want := "@@ -10,5 +20,8 @@\n20: +added line\n21:  context line\n-deleted line\n22: +another added"
	if got != want {
		t.Errorf("numberDiffLines:\ngot:  %q\nwant: %q", got, want)
	}
}

func TestNumberDiffLines_MultipleHunks(t *testing.T) {
	patch := "@@ -1,2 +1,2 @@\n+first\n ctx\n@@ -50,2 +50,2 @@\n+second\n ctx"
	got := numberDiffLines(patch)
	want := "@@ -1,2 +1,2 @@\n1: +first\n2:  ctx\n@@ -50,2 +50,2 @@\n50: +second\n51:  ctx"
	if got != want {
		t.Errorf("numberDiffLines:\ngot:  %q\nwant: %q", got, want)
	}
}

func TestBuildPrompt_EmptyPatch(t *testing.T) {
	diffs := []FileDiff{
		{Filename: "empty.go", Status: "modified", Patch: ""},
	}
	_, user := BuildPrompt(diffs, "", "", "", "", false)

	if strings.Contains(user, "empty.go") {
		t.Error("empty patch (non-too-large) should be skipped")
	}
}
