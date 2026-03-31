package review

import (
	"fmt"
	"strings"

	"github.com/getkaze/mole/internal/llm"
	"github.com/getkaze/mole/internal/personality"
)

type FormattedReview struct {
	Body     string
	Comments []FormattedComment
}

type FormattedComment struct {
	File string
	Line int
	Body string
}

// Format renders a review response using the personality engine and score.
func Format(resp *llm.ReviewResponse, engine *personality.Engine, score int) *FormattedReview {
	var body strings.Builder

	// Header
	body.WriteString(fmt.Sprintf("## %s\n\n", engine.ReviewHeader()))

	// Score badge
	body.WriteString(engine.ScoreBadge(score))
	body.WriteString("\n\n")

	// Summary with personality
	body.WriteString(engine.Summary(score, len(resp.Comments)))
	body.WriteString("\n\n")

	// Diagrams
	for _, diagram := range resp.Diagrams {
		body.WriteString("### Diagram\n\n")
		fmt.Fprintf(&body, "```mermaid\n%s\n```\n\n", diagram)
	}

	// Issues
	if len(resp.Comments) > 0 {
		body.WriteString("### Issues\n\n")
		for i, c := range resp.Comments {
			badge := engine.SeverityBadge(c.Severity)
			label := engine.SeverityLabel(c.Severity)
			cat := c.Category
			if c.Subcategory != "" {
				cat = fmt.Sprintf("%s / %s", c.Category, c.Subcategory)
			}
			fmt.Fprintf(&body, "**%d.** `%s:%d` — %s **%s** · %s\n> %s\n\n",
				i+1, c.File, c.Line, badge, label, cat, c.Message)
		}
	}

	// Inline comments
	comments := make([]FormattedComment, 0, len(resp.Comments))
	for _, c := range resp.Comments {
		badge := engine.SeverityBadge(c.Severity)
		label := engine.SeverityLabel(c.Severity)
		cat := c.Category
		if c.Subcategory != "" {
			cat = fmt.Sprintf("%s / %s", c.Category, c.Subcategory)
		}
		commentBody := fmt.Sprintf("%s **%s** · `%s`\n\n---\n%s", badge, label, cat, c.Message)
		comments = append(comments, FormattedComment{
			File: c.File,
			Line: c.Line,
			Body: commentBody,
		})
	}

	return &FormattedReview{
		Body:     body.String(),
		Comments: comments,
	}
}
