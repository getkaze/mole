package review

import (
	"fmt"
	"strings"

	"github.com/getkaze/kite/internal/i18n"
	"github.com/getkaze/kite/internal/llm"
)

var severityBadge = map[string]string{
	"must-fix":   "🟥",
	"should-fix": "🟠",
	"consider":   "🔵",
}

// severityLabel returns the localized label for a severity level.
func severityLabel(severity string, msgs i18n.Messages) string {
	switch severity {
	case "must-fix":
		return msgs.MustFix
	case "should-fix":
		return msgs.ShouldFix
	case "consider":
		return msgs.Consider
	default:
		return msgs.Consider
	}
}

type FormattedReview struct {
	Body     string
	Comments []FormattedComment
}

type FormattedComment struct {
	File string
	Line int
	Body string
}

func Format(resp *llm.ReviewResponse, lang string) *FormattedReview {
	msgs := i18n.Get(lang)
	var body strings.Builder

	body.WriteString(fmt.Sprintf("## %s\n\n", msgs.ReviewHeader))
	body.WriteString(fmt.Sprintf("### %s\n\n", msgs.Summary))
	body.WriteString(resp.Summary)
	body.WriteString("\n\n")

	for _, diagram := range resp.Diagrams {
		body.WriteString(fmt.Sprintf("### %s\n\n", msgs.DiagramHeader))
		fmt.Fprintf(&body, "```mermaid\n%s\n```\n\n", diagram)
	}

	if len(resp.Comments) > 0 {
		body.WriteString(fmt.Sprintf("### %s\n\n", msgs.IssuesFound))
		for i, c := range resp.Comments {
			badge := severityBadge[c.Severity]
			if badge == "" {
				badge = severityBadge["consider"]
			}
			label := severityLabel(c.Severity, msgs)
			fmt.Fprintf(&body, "**%d.** `%s:%d` — %s **%s** · %s\n> %s\n\n",
				i+1, c.File, c.Line, badge, label, c.Category, c.Message)
		}
	}

	if len(resp.Suggestions) > 0 {
		body.WriteString(fmt.Sprintf("### %s\n\n", msgs.Suggestions))
		for _, s := range resp.Suggestions {
			fmt.Fprintf(&body, "- %s\n", s)
		}
		body.WriteString("\n")
	}

	if len(resp.Comments) == 0 {
		body.WriteString(msgs.NoIssues)
		body.WriteString("\n")
	}

	comments := make([]FormattedComment, 0, len(resp.Comments))
	for _, c := range resp.Comments {
		badge := severityBadge[c.Severity]
		if badge == "" {
			badge = severityBadge["consider"]
		}
		label := severityLabel(c.Severity, msgs)
		commentBody := fmt.Sprintf("%s **%s** · `%s`\n\n---\n%s", badge, label, c.Category, c.Message)
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
