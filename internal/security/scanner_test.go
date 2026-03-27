package security

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScan_SQLConcat(t *testing.T) {
	dir := t.TempDir()
	code := `package main

import "database/sql"

func bad(db *sql.DB, userID string) {
	db.Query("SELECT * FROM users WHERE id=" + userID)
}
`
	os.WriteFile(filepath.Join(dir, "main.go"), []byte(code), 0o644)

	comments := Scan(dir)
	if len(comments) == 0 {
		t.Fatal("expected SQL injection finding")
	}
	if comments[0].Category != "Security" {
		t.Errorf("category = %q, want Security", comments[0].Category)
	}
	if comments[0].Subcategory != "SQL Injection" {
		t.Errorf("subcategory = %q, want SQL Injection", comments[0].Subcategory)
	}
	if comments[0].Severity != "critical" {
		t.Errorf("severity = %q, want critical", comments[0].Severity)
	}
}

func TestScan_ParameterizedQuery(t *testing.T) {
	dir := t.TempDir()
	code := `package main

import "database/sql"

func good(db *sql.DB, userID string) {
	db.Query("SELECT * FROM users WHERE id = ?", userID)
}
`
	os.WriteFile(filepath.Join(dir, "main.go"), []byte(code), 0o644)

	comments := Scan(dir)
	if len(comments) != 0 {
		t.Errorf("got %d findings, want 0 for parameterized query", len(comments))
	}
}

func TestScan_HardcodedSecret(t *testing.T) {
	dir := t.TempDir()
	code := `package main

var apiKey = "sk-ant-1234567890abcdef1234567890"
`
	os.WriteFile(filepath.Join(dir, "main.go"), []byte(code), 0o644)

	comments := Scan(dir)
	if len(comments) == 0 {
		t.Fatal("expected hardcoded secret finding")
	}
	if comments[0].Subcategory != "Secrets Exposure" {
		t.Errorf("subcategory = %q, want Secrets Exposure", comments[0].Subcategory)
	}
}

func TestScan_NoSecrets(t *testing.T) {
	dir := t.TempDir()
	code := `package main

import "os"

var apiKey = os.Getenv("API_KEY")
`
	os.WriteFile(filepath.Join(dir, "main.go"), []byte(code), 0o644)

	comments := Scan(dir)
	if len(comments) != 0 {
		t.Errorf("got %d findings, want 0 for env var usage", len(comments))
	}
}

func TestScan_ExecCommand(t *testing.T) {
	dir := t.TempDir()
	code := `package main

import "os/exec"

func run(cmd string) {
	exec.Command(cmd)
}
`
	os.WriteFile(filepath.Join(dir, "main.go"), []byte(code), 0o644)

	comments := Scan(dir)
	if len(comments) == 0 {
		t.Fatal("expected command injection finding")
	}
	if comments[0].Severity != "critical" {
		t.Errorf("severity = %q, want critical", comments[0].Severity)
	}
}

func TestScan_ExecCommandLiteral(t *testing.T) {
	dir := t.TempDir()
	code := `package main

import "os/exec"

func run() {
	exec.Command("ls", "-la")
}
`
	os.WriteFile(filepath.Join(dir, "main.go"), []byte(code), 0o644)

	comments := Scan(dir)
	if len(comments) != 0 {
		t.Errorf("got %d findings, want 0 for literal exec args", len(comments))
	}
}

func TestScan_SkipsTestFiles(t *testing.T) {
	dir := t.TempDir()
	code := `package main

import "database/sql"

func testBad(db *sql.DB, id string) {
	db.Query("SELECT * FROM users WHERE id=" + id)
}
`
	os.WriteFile(filepath.Join(dir, "main_test.go"), []byte(code), 0o644)

	comments := Scan(dir)
	if len(comments) != 0 {
		t.Errorf("got %d findings, want 0 (test files should be skipped)", len(comments))
	}
}
