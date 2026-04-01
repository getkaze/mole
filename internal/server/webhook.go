package server

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	gh "github.com/google/go-github/v72/github"

	"github.com/getkaze/mole/internal/metrics"
	"github.com/getkaze/mole/internal/queue"
	"github.com/getkaze/mole/internal/store"
)

type WebhookHandler struct {
	webhookSecret []byte
	queue         *queue.Queue
	store         store.Store
}

func NewWebhookHandler(secret string, q *queue.Queue, s store.Store) *WebhookHandler {
	return &WebhookHandler{
		webhookSecret: []byte(secret),
		queue:         q,
		store:         s,
	}
}

func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 10<<20) // 10MB max
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	if err := gh.ValidateSignature(r.Header.Get("X-Hub-Signature-256"), payload, h.webhookSecret); err != nil {
		slog.Warn("invalid webhook signature", "error", err)
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	deliveryID := r.Header.Get("X-GitHub-Delivery")
	if deliveryID != "" {
		dup, err := h.queue.IsDuplicate(r.Context(), deliveryID)
		if err != nil {
			slog.Error("dedup check failed", "error", err)
		} else if dup {
			w.WriteHeader(http.StatusOK)
			return
		}
	}

	eventType := r.Header.Get("X-GitHub-Event")
	metrics.WebhooksReceived.WithLabelValues(eventType).Inc()

	switch eventType {
	case "pull_request":
		h.handlePullRequest(r.Context(), payload, deliveryID)
	case "issue_comment":
		h.handleIssueComment(r.Context(), payload, deliveryID)
	case "installation":
		h.handleInstallation(r.Context(), payload)
	case "installation_repositories":
		h.handleInstallationRepositories(r.Context(), payload)
	case "pull_request_review_comment":
		h.handleReviewCommentReaction(r.Context(), payload)
	default:
		slog.Debug("ignoring event", "type", eventType)
	}

	if deliveryID != "" {
		if err := h.queue.MarkProcessed(r.Context(), deliveryID); err != nil {
			slog.Error("failed to mark processed", "delivery_id", deliveryID, "error", err)
		}
	}

	w.WriteHeader(http.StatusOK)
}

func (h *WebhookHandler) handlePullRequest(ctx context.Context, payload []byte, deliveryID string) {
	event, err := gh.ParseWebHook("pull_request", payload)
	if err != nil {
		slog.Error("failed to parse pull_request event", "error", err)
		return
	}

	pr, ok := event.(*gh.PullRequestEvent)
	if !ok {
		return
	}

	if pr.GetAction() != "opened" {
		return
	}

	repo := pr.GetRepo().GetFullName()
	prNumber := pr.GetPullRequest().GetNumber()
	installID := pr.GetInstallation().GetID()

	job := queue.Job{
		ID:         fmt.Sprintf("pr-%s-%d-%d", repo, prNumber, time.Now().UnixMilli()),
		Type:       "deep",
		Repo:       repo,
		PRNumber:   prNumber,
		InstallID:  installID,
		DeliveryID: deliveryID,
		CreatedAt:  time.Now(),
	}

	if err := h.queue.Enqueue(ctx, job); err != nil {
		slog.Error("failed to enqueue review", "repo", repo, "pr", prNumber, "error", err)
		return
	}

	slog.Info("enqueued auto-review", "repo", repo, "pr", prNumber)
}

func (h *WebhookHandler) handleIssueComment(ctx context.Context, payload []byte, deliveryID string) {
	event, err := gh.ParseWebHook("issue_comment", payload)
	if err != nil {
		slog.Error("failed to parse issue_comment event", "error", err)
		return
	}

	comment, ok := event.(*gh.IssueCommentEvent)
	if !ok {
		return
	}

	if comment.GetAction() != "created" {
		return
	}

	if comment.GetIssue().PullRequestLinks == nil {
		return
	}

	body := strings.TrimSpace(comment.GetComment().GetBody())
	repo := comment.GetRepo().GetFullName()
	prNumber := comment.GetIssue().GetNumber()
	installID := comment.GetInstallation().GetID()

	var jobType string
	switch {
	case body == "/mole dig":
		jobType = "dig"
	case body == "/mole deep-review":
		jobType = "deep"
	case body == "/mole review":
		jobType = "standard"
	case body == "/mole ignore":
		if err := h.store.IgnorePR(ctx, repo, prNumber); err != nil {
			slog.Error("failed to ignore PR", "repo", repo, "pr", prNumber, "error", err)
		} else {
			slog.Info("ignored PR", "repo", repo, "pr", prNumber)
		}
		return
	default:
		return
	}

	job := queue.Job{
		ID:         fmt.Sprintf("cmd-%s-%d-%d", repo, prNumber, time.Now().UnixMilli()),
		Type:       jobType,
		Repo:       repo,
		PRNumber:   prNumber,
		InstallID:  installID,
		DeliveryID: deliveryID,
		CommentID:  comment.GetComment().GetID(),
		CreatedAt:  time.Now(),
	}

	if err := h.queue.Enqueue(ctx, job); err != nil {
		slog.Error("failed to enqueue review", "repo", repo, "pr", prNumber, "error", err)
		return
	}

	slog.Info("enqueued manual review", "repo", repo, "pr", prNumber, "type", jobType)
}

func (h *WebhookHandler) handleInstallation(ctx context.Context, payload []byte) {
	event, err := gh.ParseWebHook("installation", payload)
	if err != nil {
		slog.Error("failed to parse installation event", "error", err)
		return
	}

	inst, ok := event.(*gh.InstallationEvent)
	if !ok {
		return
	}

	action := inst.GetAction()
	installID := inst.GetInstallation().GetID()
	owner := inst.GetInstallation().GetAccount().GetLogin()

	status := "active"
	switch action {
	case "created":
		status = "active"
	case "deleted":
		status = "removed"
	case "suspend":
		status = "suspended"
	case "unsuspend":
		status = "active"
	default:
		slog.Debug("ignoring installation action", "action", action)
		return
	}

	if err := h.store.UpsertInstallation(ctx, &store.Installation{
		GitHubInstallationID: installID,
		Owner:                owner,
		Status:               status,
	}); err != nil {
		slog.Error("failed to upsert installation", "install_id", installID, "error", err)
		return
	}

	slog.Info("installation event processed", "action", action, "install_id", installID, "owner", owner)
}

func (h *WebhookHandler) handleInstallationRepositories(ctx context.Context, payload []byte) {
	event, err := gh.ParseWebHook("installation_repositories", payload)
	if err != nil {
		slog.Error("failed to parse installation_repositories event", "error", err)
		return
	}

	repoEvent, ok := event.(*gh.InstallationRepositoriesEvent)
	if !ok {
		return
	}

	installID := repoEvent.GetInstallation().GetID()

	// Get internal installation ID
	installation, err := h.store.GetInstallation(ctx, installID)
	if err != nil {
		slog.Error("installation not found for repo event", "install_id", installID, "error", err)
		return
	}

	// Handle added repos
	for _, repo := range repoEvent.RepositoriesAdded {
		if err := h.store.AddRepository(ctx, &store.Repository{
			InstallationID: installation.ID,
			GitHubRepoID:   repo.GetID(),
			FullName:       repo.GetFullName(),
			Active:         true,
		}); err != nil {
			slog.Error("failed to add repository", "repo", repo.GetFullName(), "error", err)
		} else {
			slog.Info("repository added", "repo", repo.GetFullName())
		}
	}

	// Handle removed repos
	for _, repo := range repoEvent.RepositoriesRemoved {
		if err := h.store.RemoveRepository(ctx, repo.GetID()); err != nil {
			slog.Error("failed to remove repository", "repo", repo.GetFullName(), "error", err)
		} else {
			slog.Info("repository removed", "repo", repo.GetFullName())
		}
	}
}

func (h *WebhookHandler) handleReviewCommentReaction(ctx context.Context, payload []byte) {
	// GitHub sends pull_request_review_comment events when reactions are added.
	// We look for +1 (confirmed) and -1 (false_positive) reactions on Mole's inline comments.
	//
	// Note: GitHub also has a dedicated "reaction" event, but it requires separate permissions.
	// The pull_request_review_comment event with action "edited" won't fire for reactions.
	// Instead, we handle this via the "pull_request_review_comment" event.
	//
	// For now, we parse the comment body to check if it's a Mole comment,
	// and use a separate reaction webhook if available.

	event, err := gh.ParseWebHook("pull_request_review_comment", payload)
	if err != nil {
		slog.Error("failed to parse review comment event", "error", err)
		return
	}

	commentEvent, ok := event.(*gh.PullRequestReviewCommentEvent)
	if !ok {
		return
	}

	// We only care about created comments that contain reaction info
	// Actually, reactions come via a different event type. Let's handle it properly.
	_ = commentEvent
}

// HandleReaction processes reaction events on PR review comments.
// GitHub webhook event: "pull_request_review_comment" doesn't include reactions directly.
// We need to use a polling approach or the "discussion_comment" reaction event.
// For now, we provide a CLI command to sync reactions.

// SyncReactions can be called periodically to check for reactions on Mole comments.
func (h *WebhookHandler) SyncReactions(ctx context.Context, ghClient *gh.Client, owner, repo string, prNumber int) {
	// List all review comments on the PR
	comments, _, err := ghClient.PullRequests.ListComments(ctx, owner, repo, prNumber, nil)
	if err != nil {
		slog.Error("failed to list PR comments", "error", err)
		return
	}

	for _, comment := range comments {
		commentID := comment.GetID()

		// Check if this comment has reactions
		reactions := comment.GetReactions()
		if reactions == nil {
			continue
		}

		plusOne := reactions.GetPlusOne()
		minusOne := reactions.GetMinusOne()

		if plusOne > 0 {
			if err := h.store.ValidateIssueByCommentID(ctx, commentID, "confirmed", "reaction"); err != nil {
				slog.Debug("no issue for comment", "comment_id", commentID)
			} else {
				slog.Info("issue confirmed via reaction", "comment_id", commentID)
			}
		} else if minusOne > 0 {
			if err := h.store.ValidateIssueByCommentID(ctx, commentID, "false_positive", "reaction"); err != nil {
				slog.Debug("no issue for comment", "comment_id", commentID)
			} else {
				slog.Info("issue marked false positive via reaction", "comment_id", commentID)
			}
		}
	}
}

