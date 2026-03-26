package llm

import (
	"strings"
	"testing"
)

func TestBuildPrompt_Standard(t *testing.T) {
	diffs := []FileDiff{
		{Filename: "main.go", Status: "modified", Patch: "+new line"},
	}
	system, user := BuildPrompt(diffs, "", false)

	if !strings.Contains(system, "Kite") {
		t.Error("system prompt should mention Kite")
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
	system, _ := BuildPrompt(diffs, "", true)

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
	_, user := BuildPrompt(diffs, "project rules here", false)

	if !strings.Contains(user, "Project Context") {
		t.Error("should include project context header")
	}
	if !strings.Contains(user, "project rules here") {
		t.Error("should include project context content")
	}
}

func TestBuildPrompt_TooLargeFile(t *testing.T) {
	diffs := []FileDiff{
		{Filename: "big.go", Status: "modified", Patch: "", TooLarge: true},
	}
	_, user := BuildPrompt(diffs, "", false)

	if !strings.Contains(user, "too large to display") {
		t.Error("should note too-large file")
	}
}

func TestBuildPrompt_EmptyPatch(t *testing.T) {
	diffs := []FileDiff{
		{Filename: "empty.go", Status: "modified", Patch: ""},
	}
	_, user := BuildPrompt(diffs, "", false)

	if strings.Contains(user, "empty.go") {
		t.Error("empty patch (non-too-large) should be skipped")
	}
}
