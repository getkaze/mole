package ast

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateClassDiagram_StructsAndInterface(t *testing.T) {
	dir := t.TempDir()
	code := `package myapp

type Store interface {
	Save(data string) error
	Get(id int) (string, error)
}

type MySQLStore struct {
	db   string
	host string
}

func (s *MySQLStore) Save(data string) error { return nil }
func (s *MySQLStore) Get(id int) (string, error) { return "", nil }

type Config struct {
	Port int
	Host string
}
`
	os.WriteFile(filepath.Join(dir, "app.go"), []byte(code), 0o644)

	diagram, err := GenerateClassDiagram(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(diagram, "classDiagram") {
		t.Error("should start with classDiagram")
	}
	if !strings.Contains(diagram, "class Store") {
		t.Error("should contain Store interface")
	}
	if !strings.Contains(diagram, "<<interface>>") {
		t.Error("should mark Store as interface")
	}
	if !strings.Contains(diagram, "class MySQLStore") {
		t.Error("should contain MySQLStore struct")
	}
	if !strings.Contains(diagram, "class Config") {
		t.Error("should contain Config struct")
	}
	if !strings.Contains(diagram, "+Save()") {
		t.Error("should show Save method")
	}
	if !strings.Contains(diagram, "+Get()") {
		t.Error("should show Get method")
	}
	// MySQLStore implements Store
	if !strings.Contains(diagram, "MySQLStore ..|> Store") {
		t.Error("should show MySQLStore implements Store")
	}
}

func TestGenerateClassDiagram_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	diagram, err := GenerateClassDiagram(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if diagram != "" {
		t.Errorf("expected empty diagram for empty dir, got %q", diagram)
	}
}

func TestGenerateClassDiagram_SkipsTestFiles(t *testing.T) {
	dir := t.TempDir()
	code := `package myapp

type TestHelper struct {
	Name string
}
`
	os.WriteFile(filepath.Join(dir, "helper_test.go"), []byte(code), 0o644)

	diagram, err := GenerateClassDiagram(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(diagram, "TestHelper") {
		t.Error("should not include types from test files")
	}
}

func TestGenerateClassDiagram_Fields(t *testing.T) {
	dir := t.TempDir()
	code := `package myapp

type Server struct {
	port int
	host string
}
`
	os.WriteFile(filepath.Join(dir, "server.go"), []byte(code), 0o644)

	diagram, err := GenerateClassDiagram(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(diagram, "int port") {
		t.Errorf("should show field with type, got:\n%s", diagram)
	}
	if !strings.Contains(diagram, "string host") {
		t.Errorf("should show host field, got:\n%s", diagram)
	}
}
