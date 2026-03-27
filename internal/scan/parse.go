package scan

import (
	"fmt"
	"strings"
)

// InitOutput holds the parsed LLM response for mole init.
type InitOutput struct {
	Architecture string
	Conventions  string
}

// ParseInitResponse extracts architecture and conventions docs from the LLM response.
func ParseInitResponse(raw string) (*InitOutput, error) {
	archStart := strings.Index(raw, "---ARCHITECTURE---")
	convStart := strings.Index(raw, "---CONVENTIONS---")
	endMarker := strings.Index(raw, "---END---")

	if archStart == -1 || convStart == -1 {
		return nil, fmt.Errorf("invalid response format: missing section markers")
	}

	arch := strings.TrimSpace(raw[archStart+len("---ARCHITECTURE---") : convStart])
	var conv string
	if endMarker != -1 {
		conv = strings.TrimSpace(raw[convStart+len("---CONVENTIONS---") : endMarker])
	} else {
		conv = strings.TrimSpace(raw[convStart+len("---CONVENTIONS---"):])
	}

	if arch == "" || conv == "" {
		return nil, fmt.Errorf("invalid response: empty sections")
	}

	return &InitOutput{
		Architecture: arch,
		Conventions:  conv,
	}, nil
}
