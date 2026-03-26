package review

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	ghclient "github.com/getkaze/kite/internal/github"
	"github.com/getkaze/kite/internal/llm"
	"github.com/getkaze/kite/internal/metrics"
	"github.com/getkaze/kite/internal/queue"
	"github.com/getkaze/kite/internal/store"
)

type Service struct {
	ghFactory *ghclient.ClientFactory
	provider  llm.Provider
	store     store.Store
	sonnet    string
	opus      string
}

func NewService(
	ghFactory *ghclient.ClientFactory,
	provider llm.Provider,
	s store.Store,
	sonnetModel string,
	opusModel string,
) *Service {
	return &Service{
		ghFactory: ghFactory,
		provider:  provider,
		store:     s,
		sonnet:    sonnetModel,
		opus:      opusModel,
	}
}

func (s *Service) Execute(ctx context.Context, job queue.Job) error {
	start := time.Now()

	// Check if PR is ignored
	ignored, err := s.store.IsIgnored(ctx, job.Repo, job.PRNumber)
	if err != nil {
		return fmt.Errorf("checking ignored: %w", err)
	}
	if ignored {
		slog.Info("skipping ignored PR", "repo", job.Repo, "pr", job.PRNumber)
		return nil
	}

	// Parse owner/repo
	parts := strings.SplitN(job.Repo, "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid repo format: %s", job.Repo)
	}
	owner, repo := parts[0], parts[1]

	// Get GitHub client
	gh, err := s.ghFactory.Client(job.InstallID)
	if err != nil {
		return fmt.Errorf("getting github client: %w", err)
	}

	// Add 👀 reaction to signal review started
	ghclient.AddReaction(ctx, gh, owner, repo, job.PRNumber, job.CommentID, "eyes")

	// Get PR head SHA and base ref
	headSHA, baseRef, err := ghclient.GetPRHead(ctx, gh, owner, repo, job.PRNumber)
	if err != nil {
		return fmt.Errorf("getting PR head: %w", err)
	}

	// Fetch diff
	diffs, err := ghclient.FetchDiff(ctx, gh, owner, repo, job.PRNumber)
	if err != nil {
		return fmt.Errorf("fetching diff: %w", err)
	}

	// Load context files
	ctxResult, err := ghclient.LoadContext(ctx, gh, owner, repo, baseRef)
	if err != nil {
		slog.Warn("failed to load context files, continuing without", "error", err)
		ctxResult = &ghclient.ContextResult{}
	}

	// Load per-repo config
	repoCfg, err := ghclient.LoadRepoConfig(ctx, gh, owner, repo, baseRef)
	if err != nil {
		slog.Warn("failed to load repo config, using defaults", "error", err)
		repoCfg = &ghclient.RepoConfig{Language: "en"}
	}

	// Select model
	model := s.sonnet
	deep := job.Type == "deep"
	if deep {
		model = s.opus
	}

	// Convert diffs to LLM format
	llmDiffs := make([]llm.FileDiff, len(diffs))
	for i, d := range diffs {
		llmDiffs[i] = llm.FileDiff{
			Filename: d.Filename,
			Status:   d.Status,
			Patch:    d.Patch,
			TooLarge: d.TooLarge,
		}
	}

	slog.Info("reviewing PR",
		"repo", job.Repo,
		"pr", job.PRNumber,
		"type", job.Type,
		"model", model,
		"files", len(diffs),
	)

	// Review all files in a single call
	result, err := s.provider.Review(ctx, llm.ReviewRequest{
		Diff:    llmDiffs,
		Context: ctxResult.Content,
		Model:   model,
	})
	if err != nil {
		s.saveReview(ctx, job, model, nil, err)
		return fmt.Errorf("LLM review: %w", err)
	}

	// Validate line numbers
	result.Comments = ValidateComments(result.Comments, llmDiffs)

	// Format
	formatted := Format(result, repoCfg.Language)

	// Convert to GitHub review data
	reviewData := &ghclient.ReviewData{
		Body: formatted.Body,
	}
	for _, c := range formatted.Comments {
		reviewData.Comments = append(reviewData.Comments, ghclient.ReviewComment{
			File: c.File,
			Line: c.Line,
			Body: c.Body,
		})
	}

	// Post to GitHub
	if err := ghclient.PostReview(ctx, gh, owner, repo, job.PRNumber, headSHA, reviewData); err != nil {
		s.saveReview(ctx, job, model, result, err)
		return fmt.Errorf("posting review: %w", err)
	}

	// Add ✅ reaction to signal review complete
	ghclient.AddReaction(ctx, gh, owner, repo, job.PRNumber, job.CommentID, "rocket")

	// Save review record
	s.saveReview(ctx, job, model, result, nil)

	elapsed := time.Since(start)
	metrics.ReviewDuration.WithLabelValues(job.Type).Observe(elapsed.Seconds())

	slog.Info("review posted",
		"repo", job.Repo,
		"pr", job.PRNumber,
		"type", job.Type,
		"model", model,
		"comments", len(result.Comments),
		"input_tokens", result.Usage.InputTokens,
		"output_tokens", result.Usage.OutputTokens,
		"duration_ms", elapsed.Milliseconds(),
	)

	return nil
}

func (s *Service) saveReview(ctx context.Context, job queue.Job, model string, result *llm.ReviewResponse, reviewErr error) {
	r := &store.Review{
		Repo:       job.Repo,
		PRNumber:   job.PRNumber,
		ReviewType: job.Type,
		Model:      model,
		Status:     "success",
	}

	if result != nil {
		r.InputTokens = result.Usage.InputTokens
		r.OutputTokens = result.Usage.OutputTokens
		r.Summary = result.Summary
	}

	if reviewErr != nil {
		r.Status = "failed"
		r.ErrorMessage = reviewErr.Error()
	}

	if err := s.store.SaveReview(ctx, r); err != nil {
		slog.Error("failed to save review record", "error", err)
	}

	metrics.ReviewsTotal.WithLabelValues(r.ReviewType, r.Status).Inc()
	if result != nil {
		metrics.TokensUsed.WithLabelValues(model, "input").Add(float64(result.Usage.InputTokens))
		metrics.TokensUsed.WithLabelValues(model, "output").Add(float64(result.Usage.OutputTokens))
	}
}
