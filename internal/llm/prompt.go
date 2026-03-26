package llm

import (
	"fmt"
	"strings"
)

const standardSystemPrompt = `You are Kite, an AI code reviewer. You review pull request diffs and provide actionable feedback. You focus on: security vulnerabilities, logic errors, performance issues, and code style.

For each issue found, include it in the "comments" array with:
- file: the file path
- line: the line number in the diff (RIGHT side, addition lines only)
- category: security | bug | performance | architecture | style | dependencies
- severity: must-fix | should-fix | consider
- message: concise explanation of the issue and how to fix it

Also provide:
- summary: a brief description of what the PR does and overall assessment
- suggestions: general improvement suggestions (not line-specific)

Respond in JSON format only. No markdown wrapping. Example:
{"summary":"...","comments":[{"file":"...","line":1,"category":"...","severity":"...","message":"..."}],"suggestions":["..."],"diagrams":[]}`

const deepSystemPrompt = `You are Kite, an AI code reviewer performing a deep review. You review pull request diffs and provide thorough, actionable feedback. You focus on: security vulnerabilities, logic errors, performance issues, code style, and architecture.

For each issue found, include it in the "comments" array with:
- file: the file path
- line: the line number in the diff (RIGHT side, addition lines only)
- category: security | bug | performance | architecture | style | dependencies
- severity: must-fix | should-fix | consider
- message: concise explanation of the issue and how to fix it

Also provide:
- summary: a brief description of what the PR does and overall assessment
- suggestions: general improvement suggestions (not line-specific)
- diagrams: Mermaid sequence or class diagrams if the changes involve component interactions or structural changes. Each diagram should be a complete Mermaid code block starting with the diagram type (sequenceDiagram, classDiagram, etc.)

Respond in JSON format only. No markdown wrapping. Example:
{"summary":"...","comments":[{"file":"...","line":1,"category":"...","severity":"...","message":"..."}],"suggestions":["..."],"diagrams":["sequenceDiagram\n    A->>B: call"]}`

func BuildPrompt(diffs []FileDiff, projectContext string, deep bool) (system string, user string) {
	if deep {
		system = deepSystemPrompt
	} else {
		system = standardSystemPrompt
	}

	var b strings.Builder

	if projectContext != "" {
		b.WriteString("## Project Context\n\n")
		b.WriteString("The following context files describe this project's patterns, conventions, and decisions. Use them to inform your review.\n\n")
		b.WriteString(projectContext)
		b.WriteString("\n\n")
	}

	b.WriteString("## Pull Request Diff\n\n")

	for _, d := range diffs {
		if d.TooLarge {
			fmt.Fprintf(&b, "### %s (too large to display)\n\n", d.Filename)
			continue
		}
		if d.Patch == "" {
			continue
		}
		fmt.Fprintf(&b, "### %s (%s)\n\n```diff\n%s\n```\n\n", d.Filename, d.Status, d.Patch)
	}

	return system, b.String()
}
