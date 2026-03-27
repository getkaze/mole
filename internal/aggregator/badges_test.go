package aggregator

import "testing"

func TestEvaluateBadges_FirstReview(t *testing.T) {
	badges := evaluateBadges(1, 0, nil)
	if !containsBadge(badges, "first_review") {
		t.Error("should award first_review for 1 review")
	}
}

func TestEvaluateBadges_NoReviews(t *testing.T) {
	badges := evaluateBadges(0, 0, nil)
	if len(badges) != 0 {
		t.Errorf("got %d badges, want 0 for no reviews", len(badges))
	}
}

func TestEvaluateBadges_Streak5(t *testing.T) {
	badges := evaluateBadges(5, 5, nil)
	if !containsBadge(badges, "streak_5") {
		t.Error("should award streak_5")
	}
}

func TestEvaluateBadges_Streak10(t *testing.T) {
	badges := evaluateBadges(10, 10, nil)
	if !containsBadge(badges, "streak_10") {
		t.Error("should award streak_10")
	}
	if !containsBadge(badges, "streak_5") {
		t.Error("streak_10 should also include streak_5")
	}
}

func TestEvaluateBadges_ZeroCriticalMonth(t *testing.T) {
	cats := map[string]int{"Bugs": 3, "Style": 2}
	badges := evaluateBadges(5, 0, cats)
	if !containsBadge(badges, "zero_critical_month") {
		t.Error("should award zero_critical_month when no Security issues")
	}
}

func TestEvaluateBadges_HasCritical(t *testing.T) {
	cats := map[string]int{"Security": 1}
	badges := evaluateBadges(5, 0, cats)
	if containsBadge(badges, "zero_critical_month") {
		t.Error("should NOT award zero_critical_month when Security issues exist")
	}
}

func containsBadge(badges []string, name string) bool {
	for _, b := range badges {
		if b == name {
			return true
		}
	}
	return false
}
