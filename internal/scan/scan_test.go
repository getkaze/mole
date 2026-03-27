package scan

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun_ValidRepo(t *testing.T) {
	// Use the mole repo itself as test input
	root := findRepoRoot(t)
	result, err := Run(root)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	if result.Language != "go" {
		t.Errorf("Language = %q, want go", result.Language)
	}
	if !containsStr(result.BuildFiles, "go.mod") {
		t.Error("BuildFiles should contain go.mod")
	}
	if result.Structure == "" {
		t.Error("Structure should not be empty")
	}
	if len(result.Samples) == 0 {
		t.Error("Samples should not be empty")
	}
}

func TestRun_InvalidPath(t *testing.T) {
	_, err := Run("/nonexistent/path")
	if err == nil {
		t.Error("Run() should fail for nonexistent path")
	}
}

func TestRun_NotADirectory(t *testing.T) {
	f, err := os.CreateTemp("", "mole-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.Close()

	_, err = Run(f.Name())
	if err == nil {
		t.Error("Run() should fail for a file")
	}
}

func TestFormat(t *testing.T) {
	r := &Result{
		RootPath:   "/tmp/myproject",
		Language:   "go",
		Framework:  "gin",
		BuildFiles: []string{"go.mod"},
		Structure:  "├── main.go\n",
		TestInfo:   "- Test files found: 5\n",
	}

	output := r.Format()
	if !strings.Contains(output, "go") {
		t.Error("Format should contain language")
	}
	if !strings.Contains(output, "gin") {
		t.Error("Format should contain framework")
	}
	if !strings.Contains(output, "main.go") {
		t.Error("Format should contain structure")
	}
}

func TestParseInitResponse(t *testing.T) {
	raw := `Some preamble
---ARCHITECTURE---
# Architecture
This is the arch doc.
---CONVENTIONS---
# Conventions
This is the conv doc.
---END---
trailing`

	out, err := ParseInitResponse(raw)
	if err != nil {
		t.Fatalf("ParseInitResponse() error: %v", err)
	}
	if !strings.Contains(out.Architecture, "arch doc") {
		t.Errorf("Architecture = %q, want to contain 'arch doc'", out.Architecture)
	}
	if !strings.Contains(out.Conventions, "conv doc") {
		t.Errorf("Conventions = %q, want to contain 'conv doc'", out.Conventions)
	}
}

func TestParseInitResponse_MissingMarkers(t *testing.T) {
	_, err := ParseInitResponse("just some text without markers")
	if err == nil {
		t.Error("should fail when markers are missing")
	}
}

func TestParseInitResponse_NoEnd(t *testing.T) {
	raw := `---ARCHITECTURE---
arch content
---CONVENTIONS---
conv content`

	out, err := ParseInitResponse(raw)
	if err != nil {
		t.Fatalf("should handle missing ---END---: %v", err)
	}
	if out.Architecture == "" || out.Conventions == "" {
		t.Error("should parse even without ---END---")
	}
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find repo root")
		}
		dir = parent
	}
}

func containsStr(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}
