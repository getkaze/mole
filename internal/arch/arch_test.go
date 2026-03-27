package arch

import (
	"os"
	"path/filepath"
	"testing"

	ghclient "github.com/getkaze/mole/internal/github"
)

func TestValidate_NoRules(t *testing.T) {
	comments := Validate("/tmp", nil)
	if len(comments) != 0 {
		t.Errorf("got %d comments, want 0 for nil rules", len(comments))
	}
}

func TestValidate_EmptyLayers(t *testing.T) {
	comments := Validate("/tmp", &ghclient.ArchitectureRule{})
	if len(comments) != 0 {
		t.Errorf("got %d comments, want 0 for empty layers", len(comments))
	}
}

func TestValidate_DetectsViolation(t *testing.T) {
	dir := t.TempDir()

	// Create handler that imports repository
	handlerDir := filepath.Join(dir, "internal", "handlers")
	os.MkdirAll(handlerDir, 0o755)

	handlerCode := `package handlers

import (
	"fmt"
	"myapp/internal/repository"
)

func Handle() {
	fmt.Println(repository.Get())
}
`
	os.WriteFile(filepath.Join(handlerDir, "handler.go"), []byte(handlerCode), 0o644)

	rules := &ghclient.ArchitectureRule{
		Style: "clean",
		Layers: []ghclient.Layer{
			{Name: "handlers", Path: "internal/handlers/*", CanImport: []string{"service"}},
			{Name: "service", Path: "internal/service/*", CanImport: []string{"repository"}},
			{Name: "repository", Path: "internal/repository/*", CanImport: nil},
		},
	}

	comments := Validate(dir, rules)
	if len(comments) == 0 {
		t.Fatal("expected at least one violation")
	}
	if comments[0].Category != "Architecture" {
		t.Errorf("category = %q, want Architecture", comments[0].Category)
	}
	if comments[0].Subcategory != "Layer Violation" {
		t.Errorf("subcategory = %q, want Layer Violation", comments[0].Subcategory)
	}
	if comments[0].Severity != "attention" {
		t.Errorf("severity = %q, want attention", comments[0].Severity)
	}
}

func TestValidate_AllowedImport(t *testing.T) {
	dir := t.TempDir()

	handlerDir := filepath.Join(dir, "internal", "handlers")
	os.MkdirAll(handlerDir, 0o755)

	// Handler imports service — allowed
	handlerCode := `package handlers

import "myapp/internal/service"

func Handle() {
	service.Do()
}
`
	os.WriteFile(filepath.Join(handlerDir, "handler.go"), []byte(handlerCode), 0o644)

	rules := &ghclient.ArchitectureRule{
		Layers: []ghclient.Layer{
			{Name: "handlers", Path: "internal/handlers/*", CanImport: []string{"service"}},
			{Name: "service", Path: "internal/service/*", CanImport: []string{"repository"}},
		},
	}

	comments := Validate(dir, rules)
	if len(comments) != 0 {
		t.Errorf("got %d violations, want 0 for allowed import", len(comments))
	}
}

func TestValidate_SkipsTestFiles(t *testing.T) {
	dir := t.TempDir()

	handlerDir := filepath.Join(dir, "internal", "handlers")
	os.MkdirAll(handlerDir, 0o755)

	// Test file with violation — should be ignored
	testCode := `package handlers

import "myapp/internal/repository"

func TestHandle() {
	repository.Get()
}
`
	os.WriteFile(filepath.Join(handlerDir, "handler_test.go"), []byte(testCode), 0o644)

	rules := &ghclient.ArchitectureRule{
		Layers: []ghclient.Layer{
			{Name: "handlers", Path: "internal/handlers/*", CanImport: []string{"service"}},
			{Name: "repository", Path: "internal/repository/*", CanImport: nil},
		},
	}

	comments := Validate(dir, rules)
	if len(comments) != 0 {
		t.Errorf("got %d violations, want 0 (test files should be skipped)", len(comments))
	}
}
