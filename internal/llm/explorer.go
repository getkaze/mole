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
			// If context overflow (400), ask for a summary without tools using a fresh call
			if strings.Contains(err.Error(), "prompt is too long") || strings.Contains(err.Error(), "400") {
				slog.Warn("explorer: context overflow, requesting summary from collected context",
					"turn", turnsUsed+1, "error", err)
				return e.recoverSummary(ctx, systemPrompt, messages, totalUsage, turnsUsed)
			}
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

// recoverSummary handles context overflow by extracting text from the conversation
// history and asking the model for a summary in a fresh, shorter call.
func (e *Explorer) recoverSummary(ctx context.Context, systemPrompt string, messages []anthropic.MessageParam, totalUsage TokenUsage, turnsUsed int) (*ExploreResult, error) {
	// Extract all text content the model produced across turns
	var collected strings.Builder
	for _, msg := range messages {
		if msg.Role != "assistant" {
			continue
		}
		for _, block := range msg.Content {
			if block.OfText != nil {
				collected.WriteString(block.OfText.Text)
				collected.WriteString("\n\n")
			}
		}
	}

	if collected.Len() == 0 {
		return &ExploreResult{
			TurnsUsed: turnsUsed,
			Usage:     totalUsage,
		}, nil
	}

	// Truncate to fit in a single call
	text := collected.String()
	if len(text) > 50000 {
		text = text[:50000] + "\n\n[truncated]"
	}

	resp, err := e.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     e.model,
		MaxTokens: 8192,
		System: []anthropic.TextBlockParam{
			{Text: "You are a code exploration assistant. Summarize the context you collected into a structured format suitable for a code reviewer.", Type: "text"},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(
				"Here are the notes from your exploration so far. Produce a final context summary:\n\n" + text,
			)),
		},
	})
	if err != nil {
		// If even the recovery fails, return what we have raw
		slog.Warn("explorer: recovery summary failed, returning raw context", "error", err)
		return &ExploreResult{
			Context:   text,
			TurnsUsed: turnsUsed,
			Usage:     totalUsage,
		}, nil
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
		TurnsUsed: turnsUsed,
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
