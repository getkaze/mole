package review

import (
	"fmt"

	"github.com/getkaze/kite/internal/i18n"
	"github.com/getkaze/kite/internal/llm"
)

const (
	maxUsableTokens = 150_000
	systemTokens    = 500
	charsPerToken   = 4
)

func EstimateTokens(s string) int {
	return len(s) / charsPerToken
}

type FileGroup struct {
	Files       []llm.FileDiff
	TotalTokens int
}

func SplitDiffs(diffs []llm.FileDiff, contextTokens int) []FileGroup {
	available := maxUsableTokens - contextTokens - systemTokens
	if available < 1000 {
		available = 1000
	}

	var groups []FileGroup
	var current FileGroup

	for _, d := range diffs {
		fileTokens := EstimateTokens(d.Patch)
		if fileTokens == 0 {
			fileTokens = 100 // minimum for metadata
		}

		if current.TotalTokens+fileTokens > available && len(current.Files) > 0 {
			groups = append(groups, current)
			current = FileGroup{}
		}

		current.Files = append(current.Files, d)
		current.TotalTokens += fileTokens
	}

	if len(current.Files) > 0 {
		groups = append(groups, current)
	}

	return groups
}

func AggregateResponses(responses []*llm.ReviewResponse) *llm.ReviewResponse {
	if len(responses) == 1 {
		return responses[0]
	}

	agg := &llm.ReviewResponse{
		Comments:    []llm.InlineComment{},
		Suggestions: []string{},
		Diagrams:    []string{},
	}

	var summaries []string
	totalInput := 0
	totalOutput := 0

	for _, r := range responses {
		summaries = append(summaries, r.Summary)
		agg.Comments = append(agg.Comments, r.Comments...)
		agg.Suggestions = append(agg.Suggestions, r.Suggestions...)
		agg.Diagrams = append(agg.Diagrams, r.Diagrams...)
		totalInput += r.Usage.InputTokens
		totalOutput += r.Usage.OutputTokens
	}

	agg.Summary = summaries[0]
	if len(summaries) > 1 {
		for _, s := range summaries[1:] {
			agg.Summary += "\n\n" + s
		}
	}

	agg.Usage = llm.TokenUsage{
		InputTokens:  totalInput,
		OutputTokens: totalOutput,
	}

	// Deduplicate suggestions
	seen := make(map[string]bool)
	unique := make([]string, 0, len(agg.Suggestions))
	for _, s := range agg.Suggestions {
		if !seen[s] {
			seen[s] = true
			unique = append(unique, s)
		}
	}
	agg.Suggestions = unique

	return agg
}

func SplitNote(groupCount int, lang string) string {
	if groupCount <= 1 {
		return ""
	}
	msgs := i18n.Get(lang)
	return fmt.Sprintf(msgs.SplitNote, groupCount)
}
