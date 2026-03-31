package llm

import "testing"

func TestParseResponse_ValidJSON(t *testing.T) {
	raw := `{"summary":"looks good","comments":[{"file":"a.go","line":10,"category":"Security","severity":"critical","message":"bad"}]}`
	resp, err := ParseResponse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Summary != "looks good" {
		t.Errorf("summary = %q, want %q", resp.Summary, "looks good")
	}
	if len(resp.Comments) != 1 {
		t.Fatalf("comments = %d, want 1", len(resp.Comments))
	}
	if resp.Comments[0].File != "a.go" {
		t.Errorf("comment file = %q, want %q", resp.Comments[0].File, "a.go")
	}
}

func TestParseResponse_WithCodeFence(t *testing.T) {
	raw := "```json\n{\"summary\":\"ok\",\"comments\":[]}\n```"
	resp, err := ParseResponse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Summary != "ok" {
		t.Errorf("summary = %q, want %q", resp.Summary, "ok")
	}
}

func TestParseResponse_NilFieldsInitialized(t *testing.T) {
	raw := `{"summary":"ok"}`
	resp, err := ParseResponse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Comments == nil {
		t.Error("comments should be initialized, not nil")
	}
	if resp.Diagrams == nil {
		t.Error("diagrams should be initialized, not nil")
	}
}

func TestParseResponse_InvalidJSON(t *testing.T) {
	_, err := ParseResponse("not json at all")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseResponse_WithSubcategory(t *testing.T) {
	raw := `{"summary":"ok","comments":[{"file":"a.go","line":1,"category":"Security","subcategory":"SQL Injection","severity":"critical","message":"bad"}]}`
	resp, err := ParseResponse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Comments[0].Subcategory != "SQL Injection" {
		t.Errorf("subcategory = %q, want %q", resp.Comments[0].Subcategory, "SQL Injection")
	}
}

func TestParseResponse_MissingSubcategory(t *testing.T) {
	raw := `{"summary":"ok","comments":[{"file":"a.go","line":1,"category":"Bugs","severity":"attention","message":"potential issue"}]}`
	resp, err := ParseResponse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Comments[0].Subcategory != "" {
		t.Errorf("subcategory should be empty when missing, got %q", resp.Comments[0].Subcategory)
	}
}

func TestParseResponse_Whitespace(t *testing.T) {
	raw := "   \n  {\"summary\":\"trimmed\"} \n  "
	resp, err := ParseResponse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Summary != "trimmed" {
		t.Errorf("summary = %q, want %q", resp.Summary, "trimmed")
	}
}
