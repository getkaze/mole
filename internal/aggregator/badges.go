package aggregator

// evaluateBadges determines which badges a developer has earned.
func evaluateBadges(totalReviews int, streak int, issuesByCategory map[string]int) []string {
	var badges []string

	if totalReviews >= 1 {
		badges = append(badges, "first_review")
	}

	if streak >= 5 {
		badges = append(badges, "streak_5")
	}

	if streak >= 10 {
		badges = append(badges, "streak_10")
	}

	// Zero critical in the period
	if totalReviews >= 5 && issuesByCategory["Security"] == 0 {
		badges = append(badges, "zero_critical_month")
	}

	return badges
}
