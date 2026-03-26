package llm

import (
	"context"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

type Claude struct {
	client anthropic.Client
}

func NewClaude(apiKey string) *Claude {
	client := anthropic.NewClient(option.WithAPIKey(apiKey))
	return &Claude{client: client}
}

func (c *Claude) Review(ctx context.Context, req ReviewRequest) (*ReviewResponse, error) {
	system, user := BuildPrompt(req.Diff, req.Context, req.Model != "" && req.Model != "standard")

	if req.SystemPrompt != "" {
		system = req.SystemPrompt
	}

	stream := c.client.Messages.NewStreaming(ctx, anthropic.MessageNewParams{
		Model:     req.Model,
		MaxTokens: maxTokensForModel(req.Model),
		System: []anthropic.TextBlockParam{
			{Text: system, Type: "text"},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(user)),
		},
	})

	msg := anthropic.Message{}
	for stream.Next() {
		event := stream.Current()
		msg.Accumulate(event)
	}
	if err := stream.Err(); err != nil {
		return nil, fmt.Errorf("claude API call: %w", err)
	}

	var rawText string
	for _, block := range msg.Content {
		if block.Type == "text" {
			rawText += block.Text
		}
	}

	resp, err := ParseResponse(rawText)
	if err != nil {
		return nil, err
	}

	resp.Usage = TokenUsage{
		InputTokens:  int(msg.Usage.InputTokens),
		OutputTokens: int(msg.Usage.OutputTokens),
	}

	return resp, nil
}

func maxTokensForModel(model string) int64 {
	switch {
	case contains(model, "opus"):
		return 32000
	default:
		return 16000
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstr(s, substr)
}

func searchSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
