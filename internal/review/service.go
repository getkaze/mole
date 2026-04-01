package review

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"github.com/getkaze/mole/internal/git"
	ghclient "github.com/getkaze/mole/internal/github"
	"github.com/getkaze/mole/internal/llm"
	"github.com/getkaze/mole/internal/metrics"
	"github.com/getkaze/mole/internal/personality"
	"github.com/getkaze/mole/internal/queue"
	"github.com/getkaze/mole/internal/score"
	"github.com/getkaze/mole/internal/store"
)

type Service struct {
	gatewayFactory ghclient.GatewayFactory
	provider       llm.Provider
	explorer       *llm.Explorer
	repoManager    *git.RepoManager
	store          store.Store
	sonnet         string
	opus           string
	defLanguage    string
	defPersonality string
}

func NewService(
	gatewayFactory ghclient.GatewayFactory,
	provider llm.Provider,
	explorer *llm.Explorer,
	repoManager *git.RepoManager,
	s store.Store,
	sonnetModel string,
	opusModel string,
	defaultLanguage string,
	defaultPersonality string,
) *Service {
	return &Service{
		gatewayFactory: gatewayFactory,
		provider:       provider,
		explorer:       explorer,
		repoManager:    repoManager,
		store:          s,
		sonnet:         sonnetModel,
		opus:           opusModel,
		defLanguage:    defaultLanguage,
		defPersonality: defaultPersonality,
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

	// Create gateway for this job's installation
	gw := s.gatewayFactory(job.InstallID)

	// Add 👀 reaction to signal review started
	gw.AddReaction(ctx, job.Repo, job.PRNumber, job.CommentID, "eyes")

	// Get PR info (head SHA, base ref, author)
	prInfo, err := gw.GetPRInfo(ctx, job.Repo, job.PRNumber)
	if err != nil {
		return fmt.Errorf("getting PR info: %w", err)
	}
	headSHA := prInfo.HeadSHA
	baseRef := prInfo.BaseRef

	// Fetch diff
	diffs, err := gw.FetchDiff(ctx, job.Repo, job.PRNumber)
	if err != nil {
		return fmt.Errorf("fetching diff: %w", err)
	}

	// Load context files
	ctxResult, err := gw.LoadContext(ctx, job.Repo, baseRef)
	if err != nil {
		slog.Warn("failed to load context files, continuing without", "error", err)
		ctxResult = &ghclient.ContextResult{}
	}

	// Load per-repo config
	repoCfg, err := gw.LoadRepoConfig(ctx, job.Repo, baseRef)
	if err != nil {
		slog.Warn("failed to load repo config, using defaults", "error", err)
		repoCfg = &ghclient.RepoConfig{}
	}
	repoCfg.ApplyDefaults(s.defLanguage, s.defPersonality)

	// Build personality engine from repo config (needed by exploration messages and review formatting)
	engine := personality.New(repoCfg.Personality, repoCfg.Language)

	// Convert diffs to LLM format (needed by both exploration and review)
	llmDiffs := make([]llm.FileDiff, len(diffs))
	for i, d := range diffs {
		llmDiffs[i] = llm.FileDiff{
			Filename: d.Filename,
			Status:   d.Status,
			Patch:    d.Patch,
			TooLarge: d.TooLarge,
		}
	}

	// Exploration + static analysis stage: only runs for "dig" command
	dig := job.Type == "dig"
	var explorationContext string
	var staticResult *StaticAnalysisResult
	if dig && s.repoManager != nil && s.explorer != nil && s.repoManager.Enabled() {
		wtPath, firstClone, prepErr := s.repoManager.Prepare(ctx, job.Repo, prInfo.HeadRef, job.InstallID)
		if prepErr != nil {
			slog.Warn("exploration: prepare failed, continuing without",
				"repo", job.Repo, "error", prepErr)
			// Post a comment so the user knows why there's no contextual review
			if _, err := gw.PostComment(ctx, job.Repo, job.PRNumber,
				engine.ExploreCloneFail()); err != nil {
				slog.Warn("exploration: failed to post clone failure comment", "error", err)
			}
		} else if wtPath != "" {
			defer s.repoManager.Cleanup(wtPath)

			// Post/update clone comment if first clone
			var cloneCommentID int64
			if firstClone {
				var commentErr error
				cloneCommentID, commentErr = gw.PostComment(ctx, job.Repo, job.PRNumber,
					engine.ExploreCloning())
				if commentErr != nil {
					slog.Warn("exploration: failed to post clone comment", "error", commentErr)
				}
			}

			if cloneCommentID > 0 {
				if err := gw.EditComment(ctx, job.Repo, job.PRNumber, cloneCommentID,
					engine.ExploreCloned()); err != nil {
					slog.Warn("exploration: failed to update clone comment", "error", err)
				}
			}

			// Generate file tree
			tree := llm.BuildTree(wtPath, 4)

			// Run exploration
			exploreResult, exploreErr := s.explorer.Explore(ctx, llm.ExploreRequest{
				Diff:         llmDiffs,
				Tree:         tree,
				WorktreePath: wtPath,
				Language:     repoCfg.Language,
			})
			if exploreErr != nil {
				slog.Warn("exploration: explore failed, continuing without",
					"repo", job.Repo, "error", exploreErr)
			} else {
				explorationContext = llm.FormatExplorationContext(exploreResult)
				slog.Info("exploration complete",
					"repo", job.Repo,
					"pr", job.PRNumber,
					"turns", exploreResult.TurnsUsed,
					"context_bytes", len(explorationContext),
					"input_tokens", exploreResult.Usage.InputTokens,
					"output_tokens", exploreResult.Usage.OutputTokens,
				)
			}

			// Run static analysis (AST)
			deep := job.Type == "deep" || dig
			staticResult = RunStaticAnalysis(wtPath, repoCfg, deep)
			slog.Info("static analysis complete",
				"repo", job.Repo,
				"pr", job.PRNumber,
				"comments", len(staticResult.Comments),
				"diagrams", len(staticResult.Diagrams),
			)
		}
	} else if dig && (s.repoManager == nil || !s.repoManager.Enabled()) {
		slog.Warn("exploration: dig requested but disabled (base_path not configured or git not available)")
	}

	// Select model — dig uses Opus (like deep)
	model := s.sonnet
	if job.Type == "deep" || dig {
		model = s.opus
	}

	// Load previous issues for this PR to avoid duplicates
	var previousIssues string
	prevIssues, err := s.store.GetIssuesByPR(ctx, job.Repo, job.PRNumber)
	if err == nil && len(prevIssues) > 0 {
		previousIssues = formatPreviousIssues(prevIssues)
		slog.Info("loaded previous issues", "repo", job.Repo, "pr", job.PRNumber, "count", len(prevIssues))
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
		Diff:           llmDiffs,
		Context:        ctxResult.Content + explorationContext,
		Instructions:   repoCfg.Instructions,
		PreviousIssues: previousIssues,
		Model:          model,
		Language:       repoCfg.Language,
	})
	if err != nil {
		s.saveReview(ctx, job, model, prInfo.Author, nil, nil, nil, err)
		return fmt.Errorf("LLM review: %w", err)
	}

	// Merge static analysis results (AST) into LLM review
	MergeStaticAnalysis(result, staticResult)

	// Validate line numbers
	result.Comments = ValidateComments(result.Comments, llmDiffs)

	// Apply filters from repo config
	result.Comments = FilterComments(result.Comments, repoCfg.MinSeverity, repoCfg.Ignore, repoCfg.MaxInlineComments)

	// Calculate score
	scoreComments := make([]score.Comment, len(result.Comments))
	for i, c := range result.Comments {
		scoreComments[i] = score.Comment{Severity: c.Severity}
	}
	prScore := score.Calculate(scoreComments)

	// Format
	formatted := Format(result, engine, prScore)

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

	// Post review
	postResult, err := gw.PostReview(ctx, job.Repo, job.PRNumber, headSHA, reviewData)
	if err != nil {
		s.saveReview(ctx, job, model, prInfo.Author, &prScore, result, nil, err)
		return fmt.Errorf("posting review: %w", err)
	}

	// Add ✅ reaction to signal review complete
	gw.AddReaction(ctx, job.Repo, job.PRNumber, job.CommentID, "rocket")

	// Save review record and persist issues (with GitHub comment IDs)
	s.saveReview(ctx, job, model, prInfo.Author, &prScore, result, postResult, nil)

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

func (s *Service) saveReview(ctx context.Context, job queue.Job, model string, prAuthor string, prScore *int, result *llm.ReviewResponse, postResult *ghclient.PostReviewResult, reviewErr error) {
	r := &store.Review{
		Repo:       job.Repo,
		PRNumber:   job.PRNumber,
		PRAuthor:   prAuthor,
		ReviewType: job.Type,
		Model:      model,
		Score:      prScore,
		Status:     "success",
	}

	if job.InstallID != 0 {
		r.InstallationID = &job.InstallID
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

	reviewID, err := s.store.SaveReview(ctx, r)
	if err != nil {
		slog.Error("failed to save review record", "error", err)
	}

	// Persist issues and link GitHub comment IDs
	if result != nil && reviewID > 0 && len(result.Comments) > 0 {
		issues := make([]store.Issue, len(result.Comments))
		for i, c := range result.Comments {
			issues[i] = store.Issue{
				ReviewID:    reviewID,
				PRAuthor:    prAuthor,
				Category:    c.Category,
				Subcategory: c.Subcategory,
				Severity:    c.Severity,
				FilePath:    c.File,
				LineNumber:  c.Line,
				Description: c.Message,
				ModuleName:  extractModule(c.File),
			}
		}
		issueIDs, err := s.store.SaveIssues(ctx, reviewID, issues)
		if err != nil {
			slog.Error("failed to save issues", "error", err)
		}

		// Link GitHub comment IDs to issues for reaction tracking
		if postResult != nil && len(postResult.CommentIDs) > 0 && len(issueIDs) > 0 {
			for i, issueID := range issueIDs {
				if i < len(postResult.CommentIDs) {
					s.store.UpdateIssueCommentID(ctx, issueID, postResult.CommentIDs[i])
				}
			}
		}
	}

	metrics.ReviewsTotal.WithLabelValues(r.ReviewType, r.Status).Inc()
	if result != nil {
		metrics.TokensUsed.WithLabelValues(model, "input").Add(float64(result.Usage.InputTokens))
		metrics.TokensUsed.WithLabelValues(model, "output").Add(float64(result.Usage.OutputTokens))
	}
}

// formatPreviousIssues builds a text summary of previously reported issues for this PR.
func formatPreviousIssues(issues []store.Issue) string {
	var b strings.Builder
	for i, issue := range issues {
		fmt.Fprintf(&b, "%d. [%s] %s / %s — %s:%d — %s\n",
			i+1, issue.Severity, issue.Category, issue.Subcategory,
			issue.FilePath, issue.LineNumber, issue.Description)
	}
	return b.String()
}

// extractModule derives a module name from a file path.
// e.g. "infra/mysql/helpers.go" → "infra/mysql"
// e.g. "taxiStatus/service.go" → "taxiStatus"
// e.g. "main.go" → "" (root level, no module)
func extractModule(filePath string) string {
	dir := filepath.Dir(filePath)
	if dir == "." || dir == "" {
		return ""
	}
	return dir
}
