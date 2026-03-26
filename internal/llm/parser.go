package llm

import (
	"encoding/json"
	"fmt"
	"strings"
)

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

	var resp ReviewResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		return nil, fmt.Errorf("parsing LLM response as JSON: %w\nraw response: %.500s", err, raw)
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

	return &resp, nil
}
