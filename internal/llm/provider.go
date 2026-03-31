package llm

import "context"

type Provider interface {
	Review(ctx context.Context, req ReviewRequest) (*ReviewResponse, error)
	Generate(ctx context.Context, req GenerateRequest) (string, error)
}

type GenerateRequest struct {
	System string
	User   string
	Model  string
}

type ReviewRequest struct {
	Diff           []FileDiff
	Context        string
	Instructions   string // per-repo custom instructions
	PreviousIssues string // previously reported issues on this PR
	SystemPrompt   string
	Model          string
	Language       string // en, pt-BR — language for review output
}

type FileDiff struct {
	Filename string
	Status   string
	Patch    string
	TooLarge bool
}

type ReviewResponse struct {
	Summary  string          `json:"summary"`
	Comments []InlineComment `json:"comments"`
	Diagrams []string        `json:"diagrams"`
	Usage    TokenUsage
}

type InlineComment struct {
	File        string `json:"file"`
	Line        int    `json:"line"`
	Category    string `json:"category"`    // Security, Bugs, Smells, Architecture, Performance, Style
	Subcategory string `json:"subcategory"` // e.g. SQL Injection, Race Condition, Deep Nesting
	Severity    string `json:"severity"`    // critical, attention
	Message     string `json:"message"`
}

type TokenUsage struct {
	InputTokens  int
	OutputTokens int
}
