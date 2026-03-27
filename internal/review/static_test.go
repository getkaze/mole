package review

import (
	"os"
	"path/filepath"
	"testing"

	ghclient "github.com/getkaze/mole/internal/github"
	"github.com/getkaze/mole/internal/llm"
)

func TestRunStaticAnalysis_ArchViolation(t *testing.T) {
	dir := t.TempDir()

	handlerDir := filepath.Join(dir, "internal", "handlers")
	os.MkdirAll(handlerDir, 0o755)

	code := `package handlers

import "myapp/internal/repository"

func Handle() { repository.Get() }
`
	os.WriteFile(filepath.Join(handlerDir, "handler.go"), []byte(code), 0o644)

	cfg := &ghclient.RepoConfig{
		Architecture: &ghclient.ArchitectureRule{
			Layers: []ghclient.Layer{
				{Name: "handlers", Path: "internal/handlers/*", CanImport: []string{"service"}},
				{Name: "repository", Path: "internal/repository/*", CanImport: nil},
			},
		},
	}

	result := RunStaticAnalysis(dir, cfg, false)
	if len(result.Comments) == 0 {
		t.Fatal("expected architecture violation")
	}
	if result.Comments[0].Category != "Architecture" {
		t.Errorf("category = %q, want Architecture", result.Comments[0].Category)
	}
}

func TestRunStaticAnalysis_SecurityFinding(t *testing.T) {
	dir := t.TempDir()
	code := `package main

import "database/sql"

func bad(db *sql.DB, id string) {
	db.Query("SELECT * FROM users WHERE id=" + id)
}
`
	os.WriteFile(filepath.Join(dir, "main.go"), []byte(code), 0o644)

	result := RunStaticAnalysis(dir, &ghclient.RepoConfig{}, false)
	found := false
	for _, c := range result.Comments {
		if c.Category == "Security" {
			found = true
		}
	}
	if !found {
		t.Error("expected security finding")
	}
}

func TestRunStaticAnalysis_DeepDiagram(t *testing.T) {
	dir := t.TempDir()
	code := `package myapp

type Foo struct { Name string }
`
	os.WriteFile(filepath.Join(dir, "foo.go"), []byte(code), 0o644)

	result := RunStaticAnalysis(dir, &ghclient.RepoConfig{}, true)
	if len(result.Diagrams) == 0 {
		t.Error("deep review should include AST class diagram")
	}
}

func TestRunStaticAnalysis_NotDeep(t *testing.T) {
	dir := t.TempDir()
	code := `package myapp

type Foo struct { Name string }
`
	os.WriteFile(filepath.Join(dir, "foo.go"), []byte(code), 0o644)

	result := RunStaticAnalysis(dir, &ghclient.RepoConfig{}, false)
	if len(result.Diagrams) != 0 {
		t.Error("non-deep review should not include AST diagram")
	}
}

func TestMergeStaticAnalysis(t *testing.T) {
	resp := &llm.ReviewResponse{
		Comments: []llm.InlineComment{{File: "a.go", Message: "llm"}},
		Diagrams: []string{"sequenceDiagram\n  A->>B: call"},
	}
	static := &StaticAnalysisResult{
		Comments: []llm.InlineComment{{File: "b.go", Message: "arch"}},
		Diagrams: []string{"classDiagram\n  class Foo"},
	}

	MergeStaticAnalysis(resp, static)

	if len(resp.Comments) != 2 {
		t.Errorf("comments = %d, want 2", len(resp.Comments))
	}
	if len(resp.Diagrams) != 2 {
		t.Errorf("diagrams = %d, want 2", len(resp.Diagrams))
	}
}

func TestMergeStaticAnalysis_Nil(t *testing.T) {
	resp := &llm.ReviewResponse{
		Comments: []llm.InlineComment{{File: "a.go"}},
	}
	MergeStaticAnalysis(resp, nil)
	if len(resp.Comments) != 1 {
		t.Error("nil static should not change response")
	}
}
