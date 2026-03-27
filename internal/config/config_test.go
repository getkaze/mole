package config

import (
	"os"
	"path/filepath"
	"testing"
)

func validYAML() string {
	return `
github:
  app_id: 12345
  private_key_path: /tmp/key.pem
  webhook_secret: secret123
llm:
  api_key: sk-test
mysql:
  host: localhost
  database: mole
  user: root
valkey:
  host: localhost
`
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "mole.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoad_ValidConfig(t *testing.T) {
	path := writeConfig(t, validYAML())
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.GitHub.AppID != 12345 {
		t.Errorf("AppID = %d, want 12345", cfg.GitHub.AppID)
	}
	if cfg.LLM.APIKey != "sk-test" {
		t.Errorf("APIKey = %q, want %q", cfg.LLM.APIKey, "sk-test")
	}
}

func TestLoad_Defaults(t *testing.T) {
	path := writeConfig(t, validYAML())
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.MySQL.Port != 3306 {
		t.Errorf("MySQL.Port = %d, want 3306", cfg.MySQL.Port)
	}
	if cfg.Valkey.Port != 6379 {
		t.Errorf("Valkey.Port = %d, want 6379", cfg.Valkey.Port)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("Server.Port = %d, want 8080", cfg.Server.Port)
	}
	if cfg.Worker.Count != 3 {
		t.Errorf("Worker.Count = %d, want 3", cfg.Worker.Count)
	}
	if cfg.Log.Level != "info" {
		t.Errorf("Log.Level = %q, want %q", cfg.Log.Level, "info")
	}
}

func TestLoad_MissingRequiredField(t *testing.T) {
	yaml := `
github:
  app_id: 12345
  private_key_path: /tmp/key.pem
  webhook_secret: secret123
llm:
  api_key: sk-test
mysql:
  database: mole
  user: root
valkey:
  host: localhost
`
	path := writeConfig(t, yaml)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for missing mysql.host")
	}
	if got := err.Error(); !contains(got, "mysql.host") {
		t.Errorf("error = %q, want mention of mysql.host", got)
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoad_EnvOverride(t *testing.T) {
	path := writeConfig(t, validYAML())

	t.Setenv("MOLE_LLM_API_KEY", "sk-override")
	t.Setenv("MOLE_SERVER_PORT", "9090")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.LLM.APIKey != "sk-override" {
		t.Errorf("APIKey = %q, want %q", cfg.LLM.APIKey, "sk-override")
	}
	if cfg.Server.Port != 9090 {
		t.Errorf("Server.Port = %d, want 9090", cfg.Server.Port)
	}
}

func TestMySQLConfig_DSN(t *testing.T) {
	c := MySQLConfig{Host: "db", Port: 3306, Database: "mole", User: "root", Password: "pass"}
	want := "root:pass@tcp(db:3306)/mole?parseTime=true&multiStatements=true"
	if got := c.DSN(); got != want {
		t.Errorf("DSN() = %q, want %q", got, want)
	}
}

func TestValkeyConfig_Addr(t *testing.T) {
	c := ValkeyConfig{Host: "redis", Port: 6379}
	want := "redis:6379"
	if got := c.Addr(); got != want {
		t.Errorf("Addr() = %q, want %q", got, want)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsSubstring(s, substr)
}

func containsSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
