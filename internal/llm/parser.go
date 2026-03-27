package llm

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// rawComment mirrors InlineComment but with line as json.RawMessage
// to handle LLMs returning either int (42) or string ("42").
type rawComment struct {
	File        string          `json:"file"`
	Line        json.RawMessage `json:"line"`
	Category    string          `json:"category"`
	Subcategory string          `json:"subcategory"`
	Severity    string          `json:"severity"`
	Message     string          `json:"message"`
}

type rawResponse struct {
	Summary     string       `json:"summary"`
	Comments    []rawComment `json:"comments"`
	Suggestions []string     `json:"suggestions"`
	Diagrams    []string     `json:"diagrams"`
}

func ParseResponse(raw string) (*ReviewResponse, error) {
	raw = strings.TrimSpace(raw)

	// Strip markdown code fences if the LLM wrapped the response
	if strings.HasPrefix(raw, "```") {
		lines := strings.Split(raw, "\n")
		if len(lines) > 2 {
			lines = lines[1 : len(lines)-1]
			raw = strings.Join(lines, "\n")
		}
	}

	var rr rawResponse
	if err := json.Unmarshal([]byte(raw), &rr); err != nil {
		return nil, fmt.Errorf("parsing LLM response as JSON: %w\nraw response: %.500s", err, raw)
	}

	resp := &ReviewResponse{
		Summary:     rr.Summary,
		Suggestions: rr.Suggestions,
		Diagrams:    rr.Diagrams,
	}

	for _, rc := range rr.Comments {
		line := parseLine(rc.Line)
		resp.Comments = append(resp.Comments, InlineComment{
			File:        rc.File,
			Line:        line,
			Category:    rc.Category,
			Subcategory: rc.Subcategory,
			Severity:    rc.Severity,
			Message:     rc.Message,
		})
	}

	if resp.Comments == nil {
		resp.Comments = []InlineComment{}
	}
	if resp.Suggestions == nil {
		resp.Suggestions = []string{}
	}
	if resp.Diagrams == nil {
		resp.Diagrams = []string{}
	}

	return resp, nil
}

// parseLine handles line as int (42), string ("42"), or missing/null (0).
func parseLine(data json.RawMessage) int {
	if len(data) == 0 {
		return 0
	}

	// Try int
	var n int
	if err := json.Unmarshal(data, &n); err == nil {
		return n
	}

	// Try string
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		if n, err := strconv.Atoi(s); err == nil {
			return n
		}
	}

	return 0
}
