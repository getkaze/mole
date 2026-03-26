package llm

import (
	"fmt"
	"strconv"
	"strings"
)

const standardSystemPrompt = `You are Kite, an AI code reviewer. You review pull request diffs and provide actionable feedback. You focus on: security vulnerabilities, logic errors, performance issues, and code style.

Each diff line is prefixed with its file line number (e.g. "42: +code"). Use these numbers exactly.

For each issue found, include it in the "comments" array with:
- file: the file path
- line: the exact line number shown at the start of the addition line (e.g. if the line reads "42: +code", use 42)
- category: security | bug | performance | architecture | style | dependencies
- severity: must-fix | should-fix | consider
- message: concise explanation of the issue and how to fix it

Also provide:
- summary: a brief description of what the PR does and overall assessment
- suggestions: general improvement suggestions (not line-specific)

Respond in JSON format only. No markdown wrapping. Example:
{"summary":"...","comments":[{"file":"...","line":1,"category":"...","severity":"...","message":"..."}],"suggestions":["..."],"diagrams":[]}`

const deepSystemPrompt = `You are Kite, an AI code reviewer performing a deep review. You review pull request diffs and provide thorough, actionable feedback. You focus on: security vulnerabilities, logic errors, performance issues, code style, and architecture.

Each diff line is prefixed with its file line number (e.g. "42: +code"). Use these numbers exactly.

For each issue found, include it in the "comments" array with:
- file: the file path
- line: the exact line number shown at the start of the addition line (e.g. if the line reads "42: +code", use 42)
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
		fmt.Fprintf(&b, "### %s (%s)\n\n```diff\n%s\n```\n\n", d.Filename, d.Status, numberDiffLines(d.Patch))
	}

	return system, b.String()
}

// numberDiffLines prefixes each diff line with its file line number so the LLM
// can reference exact line numbers instead of counting from hunk headers.
// Addition and context lines get "N: " prefixes; deletion and hunk header lines
// are passed through unchanged.
func numberDiffLines(patch string) string {
	lines := strings.Split(patch, "\n")
	var b strings.Builder
	b.Grow(len(patch) + len(lines)*6)

	newLine := 0
	for i, raw := range lines {
		if i > 0 {
			b.WriteByte('\n')
		}
		if strings.HasPrefix(raw, "@@") {
			newLine = parseNewStart(raw)
			b.WriteString(raw)
		} else if newLine == 0 {
			b.WriteString(raw)
		} else if strings.HasPrefix(raw, "+") {
			fmt.Fprintf(&b, "%d: %s", newLine, raw)
			newLine++
		} else if strings.HasPrefix(raw, "-") {
			b.WriteString(raw)
		} else {
			// Context line — has a line number but is not an addition
			fmt.Fprintf(&b, "%d: %s", newLine, raw)
			newLine++
		}
	}

	return b.String()
}

// parseNewStart extracts the new-file start line from a hunk header.
// e.g. "@@ -10,5 +20,8 @@" → 20
func parseNewStart(header string) int {
	parts := strings.Split(header, "+")
	if len(parts) < 2 {
		return 0
	}
	numPart := strings.Fields(parts[1])[0]
	numPart = strings.TrimRight(numPart, " @")
	if idx := strings.Index(numPart, ","); idx != -1 {
		numPart = numPart[:idx]
	}
	n, err := strconv.Atoi(numPart)
	if err != nil {
		return 0
	}
	return n
}
