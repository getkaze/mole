package review

import (
	"log/slog"
	"strconv"
	"strings"

	"github.com/getkaze/mole/internal/llm"
)

func ValidateComments(comments []llm.InlineComment, diffs []llm.FileDiff) []llm.InlineComment {
	additionLines := buildAdditionLines(diffs)
	valid := make([]llm.InlineComment, 0, len(comments))

	for _, c := range comments {
		lines, exists := additionLines[c.File]
		if !exists {
			slog.Warn("dropping comment: file not in diff", "file", c.File, "line", c.Line)
			continue
		}

		if !lines[c.Line] {
			slog.Warn("dropping comment: line is not an addition line", "file", c.File, "line", c.Line)
			continue
		}

		valid = append(valid, c)
	}

	return valid
}

// buildAdditionLines walks each diff patch and collects the exact new-side line
// numbers that are addition lines (prefixed with "+"). Only these lines are valid
// targets for GitHub inline comments.
func buildAdditionLines(diffs []llm.FileDiff) map[string]map[int]bool {
	result := make(map[string]map[int]bool)

	for _, d := range diffs {
		if d.Patch == "" {
			continue
		}
		lines := make(map[int]bool)
		newLine := 0

		for _, raw := range strings.Split(d.Patch, "\n") {
			if strings.HasPrefix(raw, "@@") {
				start, _ := parseHunkHeader(raw)
				newLine = start
				continue
			}
			if newLine == 0 {
				continue
			}

			if strings.HasPrefix(raw, "+") {
				lines[newLine] = true
				newLine++
			} else if strings.HasPrefix(raw, "-") {
				// Deletion — does not advance new-side line number
			} else {
				// Context line — advances new-side line number but is not a valid target
				newLine++
			}
		}

		result[d.Filename] = lines
	}

	return result
}

// parseHunkHeader extracts the new file start and count from a unified diff hunk header.
// Format: @@ -old_start,old_count +new_start,new_count @@
func parseHunkHeader(line string) (start int, count int) {
	parts := strings.Split(line, "+")
	if len(parts) < 2 {
		return 0, 0
	}
	// Take the part after + and before the next space or @@
	numPart := strings.Fields(parts[1])[0]
	numPart = strings.TrimRight(numPart, " @")

	if strings.Contains(numPart, ",") {
		split := strings.SplitN(numPart, ",", 2)
		s, err1 := strconv.Atoi(split[0])
		c, err2 := strconv.Atoi(split[1])
		if err1 != nil || err2 != nil {
			return 0, 0
		}
		return s, c
	}

	s, err := strconv.Atoi(numPart)
	if err != nil {
		return 0, 0
	}
	return s, 1
}
