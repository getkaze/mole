package review

import (
	"path/filepath"
	"sort"

	"github.com/getkaze/mole/internal/llm"
)

// severityRank maps severity to a numeric rank for comparison and sorting.
var severityRank = map[string]int{
	"critical":   3,
	"attention":  2,
	"suggestion": 1,
}

// FilterComments applies min severity, ignore patterns, and max comments cap.
func FilterComments(comments []llm.InlineComment, minSeverity string, ignorePatterns []string, maxComments int) []llm.InlineComment {
	minRank := severityRank[minSeverity]

	var filtered []llm.InlineComment
	for _, c := range comments {
		// Filter by severity
		if minRank > 0 && severityRank[c.Severity] < minRank {
			continue
		}

		// Filter by ignore patterns
		if matchesAny(c.File, ignorePatterns) {
			continue
		}

		filtered = append(filtered, c)
	}

	// Cap at max comments, keeping highest severity first
	if maxComments > 0 && len(filtered) > maxComments {
		sort.SliceStable(filtered, func(i, j int) bool {
			return severityRank[filtered[i].Severity] > severityRank[filtered[j].Severity]
		})
		filtered = filtered[:maxComments]
	}

	return filtered
}

func matchesAny(file string, patterns []string) bool {
	for _, p := range patterns {
		if matched, _ := filepath.Match(p, file); matched {
			return true
		}
		// Also try matching just the filename for patterns like "*_test.go"
		if matched, _ := filepath.Match(p, filepath.Base(file)); matched {
			return true
		}
	}
	return false
}
