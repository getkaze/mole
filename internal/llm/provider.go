package llm

import "context"

type Provider interface {
	Review(ctx context.Context, req ReviewRequest) (*ReviewResponse, error)
}

type ReviewRequest struct {
	Diff         []FileDiff
	Context      string
	SystemPrompt string
	Model        string
}

type FileDiff struct {
	Filename string
	Status   string
	Patch    string
	TooLarge bool
}

type ReviewResponse struct {
	Summary     string          `json:"summary"`
	Comments    []InlineComment `json:"comments"`
	Suggestions []string        `json:"suggestions"`
	Diagrams    []string        `json:"diagrams"`
	Usage       TokenUsage
}

type InlineComment struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Category string `json:"category"` // security, bug, performance, architecture, style, dependencies
	Severity string `json:"severity"` // must-fix, should-fix, consider
	Message  string `json:"message"`
}

type TokenUsage struct {
	InputTokens  int
	OutputTokens int
}
