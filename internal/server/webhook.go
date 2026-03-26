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

	"github.com/getkaze/kite/internal/metrics"
	"github.com/getkaze/kite/internal/queue"
	"github.com/getkaze/kite/internal/store"
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
		Type:       "standard",
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
	case body == "/kite deep-review":
		jobType = "deep"
	case body == "/kite review":
		jobType = "standard"
	case body == "/kite ignore":
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

