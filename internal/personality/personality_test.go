package personality

import (
	"strings"
	"testing"
)

func TestNew_DefaultsToMole(t *testing.T) {
	e := New("", "en")
	if e.mode != ModeMole {
		t.Errorf("mode = %q, want %q", e.mode, ModeMole)
	}
}

func TestNew_InvalidMode(t *testing.T) {
	e := New("unknown", "en")
	if e.mode != ModeMole {
		t.Errorf("mode = %q, want %q for invalid input", e.mode, ModeMole)
	}
}

func TestNew_ValidModes(t *testing.T) {
	for _, mode := range []string{"mole", "formal", "minimal"} {
		e := New(mode, "en")
		if string(e.mode) != mode {
			t.Errorf("mode = %q, want %q", e.mode, mode)
		}
	}
}

func TestSummary_MoleClean(t *testing.T) {
	e := New("mole", "en")
	got := e.Summary(100, 0)
	if !strings.Contains(got, "Looking good") {
		t.Errorf("mole clean summary should be enthusiastic, got %q", got)
	}
}

func TestSummary_MoleIssues(t *testing.T) {
	e := New("mole", "en")
	got := e.Summary(75, 3)
	if !strings.Contains(got, "3") || !strings.Contains(got, "75") {
		t.Errorf("mole summary should contain issue count and score, got %q", got)
	}
}

func TestSummary_FormalClean(t *testing.T) {
	e := New("formal", "en")
	got := e.Summary(100, 0)
	if strings.ContainsAny(got, "🐭🟢🎉") {
		t.Errorf("formal clean should have no emoji, got %q", got)
	}
}

func TestSummary_FormalIssues(t *testing.T) {
	e := New("formal", "en")
	got := e.Summary(60, 5)
	if !strings.Contains(got, "5") || !strings.Contains(got, "60") {
		t.Errorf("formal summary should contain count and score, got %q", got)
	}
	if strings.ContainsAny(got, "🐭🟢🎉") {
		t.Errorf("formal should have no emoji, got %q", got)
	}
}

func TestSummary_Minimal(t *testing.T) {
	e := New("minimal", "en")
	got := e.Summary(85, 2)
	if len(got) > 50 {
		t.Errorf("minimal summary should be terse, got %q (len %d)", got, len(got))
	}
}

func TestCleanPR_AllModes(t *testing.T) {
	for _, mode := range []string{"mole", "formal", "minimal"} {
		e := New(mode, "en")
		got := e.CleanPR()
		if got == "" {
			t.Errorf("CleanPR() for %s should not be empty", mode)
		}
	}
}

func TestIssuePrefix_Mole(t *testing.T) {
	e := New("mole", "en")
	crit := e.IssuePrefix("critical")
	if !strings.Contains(crit, "🔴") {
		t.Errorf("mole critical prefix should have red indicator, got %q", crit)
	}
	att := e.IssuePrefix("attention")
	if !strings.Contains(att, "🟡") {
		t.Errorf("mole attention prefix should have yellow indicator, got %q", att)
	}
}

func TestIssuePrefix_Minimal(t *testing.T) {
	e := New("minimal", "en")
	if got := e.IssuePrefix("critical"); got != "" {
		t.Errorf("minimal prefix should be empty, got %q", got)
	}
}

func TestScoreBadge_Mole(t *testing.T) {
	e := New("mole", "en")
	if got := e.ScoreBadge(95); !strings.Contains(got, "🟢") {
		t.Errorf("score 95 should be green, got %q", got)
	}
	if got := e.ScoreBadge(75); !strings.Contains(got, "🟡") {
		t.Errorf("score 75 should be yellow, got %q", got)
	}
	if got := e.ScoreBadge(50); !strings.Contains(got, "🔴") {
		t.Errorf("score 50 should be red, got %q", got)
	}
}

func TestScoreBadge_Formal(t *testing.T) {
	e := New("formal", "en")
	got := e.ScoreBadge(85)
	if !strings.Contains(got, "Quality Score") {
		t.Errorf("formal score should say 'Quality Score', got %q", got)
	}
}

func TestSeverityBadge(t *testing.T) {
	e := New("mole", "en")
	if got := e.SeverityBadge("critical"); got != "🔴" {
		t.Errorf("critical badge = %q, want 🔴", got)
	}
	if got := e.SeverityBadge("attention"); got != "🟡" {
		t.Errorf("attention badge = %q, want 🟡", got)
	}
	if got := e.SeverityBadge("suggestion"); got != "🟡" {
		t.Errorf("suggestion badge should fallback to attention = %q, want 🟡", got)
	}
}

func TestSeverityLabel_EN(t *testing.T) {
	e := New("mole", "en")
	if got := e.SeverityLabel("critical"); got != "Critical" {
		t.Errorf("label = %q, want Critical", got)
	}
}

func TestSeverityLabel_PT(t *testing.T) {
	e := New("mole", "pt-BR")
	if got := e.SeverityLabel("critical"); got != "Crítico" {
		t.Errorf("label = %q, want Crítico", got)
	}
}

func TestPortuguese_MoleSummary(t *testing.T) {
	e := New("mole", "pt-BR")
	got := e.Summary(80, 2)
	if !strings.Contains(got, "toupeira") {
		t.Errorf("pt-BR mole summary should mention toupeira, got %q", got)
	}
}
