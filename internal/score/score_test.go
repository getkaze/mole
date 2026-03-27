package score

import "testing"

func TestCalculate_NoComments(t *testing.T) {
	if got := Calculate(nil); got != 100 {
		t.Errorf("Calculate(nil) = %d, want 100", got)
	}
}

func TestCalculate_Empty(t *testing.T) {
	if got := Calculate([]Comment{}); got != 100 {
		t.Errorf("Calculate([]) = %d, want 100", got)
	}
}

func TestCalculate_Mixed(t *testing.T) {
	comments := []Comment{
		{Severity: "critical"},
		{Severity: "critical"},
		{Severity: "attention"},
		{Severity: "attention"},
		{Severity: "attention"},
		{Severity: "suggestion"},
	}
	// 100 - 30 - 15 - 1 = 54
	want := 54
	if got := Calculate(comments); got != want {
		t.Errorf("Calculate() = %d, want %d", got, want)
	}
}

func TestCalculate_FloorAtZero(t *testing.T) {
	var comments []Comment
	for i := 0; i < 20; i++ {
		comments = append(comments, Comment{Severity: "critical"})
	}
	if got := Calculate(comments); got != 0 {
		t.Errorf("Calculate() = %d, want 0 (floor)", got)
	}
}

func TestCalculate_UnknownSeverity(t *testing.T) {
	comments := []Comment{{Severity: "unknown"}}
	if got := Calculate(comments); got != 100 {
		t.Errorf("unknown severity should not penalize, got %d", got)
	}
}

func TestCalculate_OnlySuggestions(t *testing.T) {
	comments := []Comment{
		{Severity: "suggestion"},
		{Severity: "suggestion"},
		{Severity: "suggestion"},
	}
	if got := Calculate(comments); got != 97 {
		t.Errorf("3 suggestions = %d, want 97", got)
	}
}
