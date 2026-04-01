package llm

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

//go:embed agents/explorer.md
var explorerPrompt string

// Explorer runs a multi-turn Haiku conversation with tools to explore
// a repository and collect context for code review.
type Explorer struct {
	client   anthropic.Client
	maxTurns int
	model    string
}

// NewExplorer creates an Explorer for the given API key and configuration.
func NewExplorer(apiKey string, maxTurns int, model string) *Explorer {
	return &Explorer{
		client:   anthropic.NewClient(option.WithAPIKey(apiKey)),
		maxTurns: maxTurns,
		model:    model,
	}
}

// ExploreRequest contains everything the explorer needs.
type ExploreRequest struct {
	Diff         []FileDiff
	Tree         string
	WorktreePath string
	Language     string
}

// ExploreResult contains the context collected by the exploration.
type ExploreResult struct {
	Context   string     // formatted context string ready for the review prompt
	TurnsUsed int
	Usage     TokenUsage
}

// Explore runs the multi-turn tool use loop.
func (e *Explorer) Explore(ctx context.Context, req ExploreRequest) (*ExploreResult, error) {
	tools := ToolDefinitions()
	executor := NewToolExecutor(req.WorktreePath)

	systemPrompt := explorerPrompt
	if req.Language != "" && req.Language != "en" {
		systemPrompt += fmt.Sprintf("\n\nIMPORTANT: Write your final summary in %s.", req.Language)
	}

	userContent := buildExplorerUserMessage(req)

	messages := []anthropic.MessageParam{
		anthropic.NewUserMessage(anthropic.NewTextBlock(userContent)),
	}

	var totalUsage TokenUsage
	turnsUsed := 0

	for turnsUsed < e.maxTurns {
		resp, err := e.client.Messages.New(ctx, anthropic.MessageNewParams{
			Model:     e.model,
			MaxTokens: 16384,
			System: []anthropic.TextBlockParam{
				{Text: systemPrompt, Type: "text"},
			},
			Messages: messages,
			Tools:    tools,
		})
		if err != nil {
			return nil, fmt.Errorf("explorer API call (turn %d): %w", turnsUsed+1, err)
		}

		totalUsage.InputTokens += int(resp.Usage.InputTokens)
		totalUsage.OutputTokens += int(resp.Usage.OutputTokens)

		// If model didn't request tools, we're done
		if resp.StopReason != anthropic.StopReasonToolUse {
			// Extract final text response
			var finalText string
			for _, block := range resp.Content {
				if block.Type == "text" {
					finalText += block.Text
				}
			}

			return &ExploreResult{
				Context:   finalText,
				TurnsUsed: turnsUsed + 1,
				Usage:     totalUsage,
			}, nil
		}

		// Append assistant message with tool use blocks
		var assistantBlocks []anthropic.ContentBlockParamUnion
		for _, block := range resp.Content {
			assistantBlocks = append(assistantBlocks, block.ToParam())
		}
		messages = append(messages, anthropic.NewAssistantMessage(assistantBlocks...))

		// Execute each tool call and build results
		var resultBlocks []anthropic.ContentBlockParamUnion
		for _, block := range resp.Content {
			if block.Type != "tool_use" {
				continue
			}

			slog.Debug("explorer tool call",
				"tool", block.Name,
				"input", string(block.Input),
				"turn", turnsUsed+1,
			)

			result, isError := executor.Execute(block.Name, block.Input)

			slog.Debug("explorer tool result",
				"tool", block.Name,
				"is_error", isError,
				"result_bytes", len(result),
			)

			resultBlocks = append(resultBlocks, anthropic.NewToolResultBlock(
				block.ID,
				result,
				isError,
			))
		}

		messages = append(messages, anthropic.NewUserMessage(resultBlocks...))
		turnsUsed++
	}

	// Max turns reached — ask model for final summary without tools
	messages = append(messages, anthropic.NewUserMessage(
		anthropic.NewTextBlock("You have reached the maximum number of exploration turns. Please provide your final context summary now based on what you have collected so far."),
	))

	resp, err := e.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     e.model,
		MaxTokens: 16384,
		System: []anthropic.TextBlockParam{
			{Text: systemPrompt, Type: "text"},
		},
		Messages: messages,
	})
	if err != nil {
		return nil, fmt.Errorf("explorer final summary: %w", err)
	}

	totalUsage.InputTokens += int(resp.Usage.InputTokens)
	totalUsage.OutputTokens += int(resp.Usage.OutputTokens)

	var finalText string
	for _, block := range resp.Content {
		if block.Type == "text" {
			finalText += block.Text
		}
	}

	return &ExploreResult{
		Context:   finalText,
		TurnsUsed: turnsUsed + 1,
		Usage:     totalUsage,
	}, nil
}

func buildExplorerUserMessage(req ExploreRequest) string {
	var b strings.Builder

	b.WriteString("## Repository Structure\n\n```\n")
	b.WriteString(req.Tree)
	b.WriteString("```\n\n")

	b.WriteString("## Pull Request Diff\n\n")
	for _, d := range req.Diff {
		if d.TooLarge {
			fmt.Fprintf(&b, "### %s (%s) — too large, skipped\n\n", d.Filename, d.Status)
			continue
		}
		fmt.Fprintf(&b, "### %s (%s)\n\n```diff\n%s\n```\n\n", d.Filename, d.Status, d.Patch)
	}

	b.WriteString("Explore the repository to collect context that will help a senior reviewer understand the full impact of this PR. Use the tools to read relevant files, search for related code, and understand the architecture around the changed files.")

	return b.String()
}

// FormatExplorationContext wraps the explorer output into a section
// suitable for appending to the review prompt context.
func FormatExplorationContext(result *ExploreResult) string {
	if result == nil || result.Context == "" {
		return ""
	}

	var b strings.Builder
	b.WriteString("\n\n## Codebase Context (collected by automated exploration)\n\n")
	b.WriteString(result.Context)
	return b.String()
}
