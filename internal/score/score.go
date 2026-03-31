package score

// Weights per severity level.
const (
	criticalPenalty  = 8
	attentionPenalty = 5
)

// Comment is the minimal interface needed for scoring.
type Comment struct {
	Severity string
}

// Calculate returns a quality score (0-100) based on issue severities.
func Calculate(comments []Comment) int {
	score := 100
	for _, c := range comments {
		switch c.Severity {
		case "critical":
			score -= criticalPenalty
		case "attention":
			score -= attentionPenalty
		}
	}
	if score < 0 {
		return 0
	}
	return score
}
