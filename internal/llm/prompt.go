package llm

import (
	"fmt"
	"strconv"
	"strings"
)

const standardSystemPrompt = `You are Mole, an AI code reviewer. You dig deep into code to find bugs, vulnerabilities, and opportunities for improvement.

IMPORTANT: Only review lines that were ADDED or CHANGED in this PR (lines starting with "+"). Do NOT comment on pre-existing code that appears as context in the diff. If a method has a pre-existing bug but the PR didn't touch that code, do NOT flag it. Focus exclusively on what the PR introduces or modifies.

Each diff line is prefixed with its file line number (e.g. "42: +code"). Use these numbers exactly. Only use line numbers from lines that start with "+".

For each issue found, include it in the "comments" array with:
- file: the file path
- line: the exact line number shown at the start of the addition line (e.g. if the line reads "42: +code", use 42). NEVER use a line number from a context line or deletion line.
- category: one of Security | Bugs | Smells | Architecture | Performance | Style
- subcategory: specific issue type. Valid subcategories per category:
  - Security: SQL Injection, XSS, Auth Bypass, Secrets Exposure, Insecure Dependencies, CSRF, Data Exposure
  - Bugs: Null/Nil Reference, Race Condition, Unhandled Error, Resource Leak, Logic Error
  - Smells: Cyclomatic Complexity, Duplication, Poor Naming, Dead Code, Deep Nesting
  - Architecture: Layer Violation, Circular Dependency, Tight Coupling, God Class, API Breaking Change
  - Performance: N+1 Query, Unbounded Query, Missing Cache, Blocking I/O in Hot Path
  - Style: Convention Violation, Missing Documentation, Missing Tests
- severity: critical | attention | suggestion
- message: concise explanation of the issue and how to fix it

Also provide:
- summary: a brief description of what the PR does and overall assessment
- suggestions: general improvement suggestions (not line-specific)

Respond in JSON format only. No markdown wrapping. Example:
{"summary":"...","comments":[{"file":"...","line":1,"category":"Security","subcategory":"SQL Injection","severity":"critical","message":"..."}],"suggestions":["..."],"diagrams":[]}`

const deepSystemPrompt = `You are Mole, an AI code reviewer performing a deep review. You dig deep into code to find bugs, vulnerabilities, architectural issues, and opportunities for improvement.

IMPORTANT: Only review lines that were ADDED or CHANGED in this PR (lines starting with "+"). Do NOT comment on pre-existing code that appears as context in the diff. If a method has a pre-existing bug but the PR didn't touch that code, do NOT flag it. Focus exclusively on what the PR introduces or modifies.

Each diff line is prefixed with its file line number (e.g. "42: +code"). Use these numbers exactly. Only use line numbers from lines that start with "+".

For each issue found, include it in the "comments" array with:
- file: the file path
- line: the exact line number shown at the start of the addition line (e.g. if the line reads "42: +code", use 42). NEVER use a line number from a context line or deletion line.
- category: one of Security | Bugs | Smells | Architecture | Performance | Style
- subcategory: specific issue type. Valid subcategories per category:
  - Security: SQL Injection, XSS, Auth Bypass, Secrets Exposure, Insecure Dependencies, CSRF, Data Exposure
  - Bugs: Null/Nil Reference, Race Condition, Unhandled Error, Resource Leak, Logic Error
  - Smells: Cyclomatic Complexity, Duplication, Poor Naming, Dead Code, Deep Nesting
  - Architecture: Layer Violation, Circular Dependency, Tight Coupling, God Class, API Breaking Change
  - Performance: N+1 Query, Unbounded Query, Missing Cache, Blocking I/O in Hot Path
  - Style: Convention Violation, Missing Documentation, Missing Tests
- severity: critical | attention | suggestion
- message: concise explanation of the issue and how to fix it

Also provide:
- summary: a brief description of what the PR does and overall assessment
- suggestions: general improvement suggestions (not line-specific)
- diagrams: Mermaid sequence or class diagrams if the changes involve component interactions or structural changes. Each diagram should be a complete Mermaid code block starting with the diagram type (sequenceDiagram, classDiagram, etc.)

Respond in JSON format only. No markdown wrapping. Example:
{"summary":"...","comments":[{"file":"...","line":1,"category":"Security","subcategory":"SQL Injection","severity":"critical","message":"..."}],"suggestions":["..."],"diagrams":["sequenceDiagram\n    A->>B: call"]}`

func BuildPrompt(diffs []FileDiff, projectContext string, instructions string, previousIssues string, deep bool) (system string, user string) {
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

	if instructions != "" {
		b.WriteString("## Repository-Specific Instructions\n\n")
		b.WriteString("Follow these additional instructions when reviewing this code:\n\n")
		b.WriteString(instructions)
		b.WriteString("\n\n")
	}

	if previousIssues != "" {
		b.WriteString("## Previously Reported Issues\n\n")
		b.WriteString("The following issues were already reported in previous reviews of this same PR. Do NOT repeat them. Only report NEW issues not listed below.\n\n")
		b.WriteString(previousIssues)
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
