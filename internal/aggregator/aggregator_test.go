package aggregator

import (
	"testing"

	"github.com/getkaze/mole/internal/store"
)

func TestCountByField(t *testing.T) {
	issues := []store.Issue{
		{Category: "Security"},
		{Category: "Security"},
		{Category: "Bugs"},
		{Category: "Style"},
	}
	result := countByField(issues, func(i store.Issue) string { return i.Category })
	if result["Security"] != 2 {
		t.Errorf("Security = %d, want 2", result["Security"])
	}
	if result["Bugs"] != 1 {
		t.Errorf("Bugs = %d, want 1", result["Bugs"])
	}
}

func TestCountUniqueReviews(t *testing.T) {
	issues := []store.Issue{
		{ReviewID: 1},
		{ReviewID: 1},
		{ReviewID: 2},
		{ReviewID: 3},
	}
	if got := countUniqueReviews(issues); got != 3 {
		t.Errorf("unique reviews = %d, want 3", got)
	}
}

func TestCalculateStreak_AllClean(t *testing.T) {
	a := &Aggregator{}
	issues := []store.Issue{
		{ReviewID: 1, Severity: "suggestion"},
		{ReviewID: 2, Severity: "attention"},
		{ReviewID: 3, Severity: "suggestion"},
	}
	if got := a.calculateStreak(issues); got != 3 {
		t.Errorf("streak = %d, want 3", got)
	}
}

func TestCalculateStreak_BrokenByRecent(t *testing.T) {
	a := &Aggregator{}
	issues := []store.Issue{
		{ReviewID: 1, Severity: "suggestion"},
		{ReviewID: 2, Severity: "critical"},
		{ReviewID: 3, Severity: "suggestion"},
	}
	// Review 3 is clean, review 2 breaks streak
	if got := a.calculateStreak(issues); got != 1 {
		t.Errorf("streak = %d, want 1", got)
	}
}

func TestCalculateStreak_AllCritical(t *testing.T) {
	a := &Aggregator{}
	issues := []store.Issue{
		{ReviewID: 1, Severity: "critical"},
		{ReviewID: 2, Severity: "critical"},
	}
	if got := a.calculateStreak(issues); got != 0 {
		t.Errorf("streak = %d, want 0", got)
	}
}

func TestCalculateStreak_Empty(t *testing.T) {
	a := &Aggregator{}
	if got := a.calculateStreak(nil); got != 0 {
		t.Errorf("streak = %d, want 0", got)
	}
}

func TestCalculateStreak_MultipleIssuesPerReview(t *testing.T) {
	a := &Aggregator{}
	issues := []store.Issue{
		{ReviewID: 1, Severity: "suggestion"},
		{ReviewID: 1, Severity: "attention"},
		{ReviewID: 2, Severity: "suggestion"},
		{ReviewID: 3, Severity: "suggestion"},
		{ReviewID: 3, Severity: "suggestion"},
	}
	// All reviews are clean (no critical)
	if got := a.calculateStreak(issues); got != 3 {
		t.Errorf("streak = %d, want 3", got)
	}
}
