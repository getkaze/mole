package aggregator

import (
	"context"
	"log/slog"
	"strings"
	"time"

	gh "github.com/google/go-github/v72/github"

	ghclient "github.com/getkaze/mole/internal/github"
	"github.com/getkaze/mole/internal/store"
)

// ReactionSyncer polls GitHub for reactions on Mole's inline comments
// and updates issue validation status.
type ReactionSyncer struct {
	store     store.Store
	ghFactory *ghclient.ClientFactory
}

// NewReactionSyncer creates a reaction syncer.
func NewReactionSyncer(s store.Store, ghFactory *ghclient.ClientFactory) *ReactionSyncer {
	return &ReactionSyncer{store: s, ghFactory: ghFactory}
}

// Sync checks recent reviews for reactions on inline comments.
func (rs *ReactionSyncer) Sync(ctx context.Context) {
	now := time.Now()
	from := now.AddDate(0, 0, -7) // check last 7 days of reviews

	// Get all issues with github_comment_id that are still pending
	issues, err := rs.store.GetPendingValidationIssues(ctx, from, now)
	if err != nil {
		slog.Error("reaction sync: failed to get pending issues", "error", err)
		return
	}

	if len(issues) == 0 {
		return
	}

	// Group issues by repo + PR for efficient API calls
	type prKey struct {
		owner    string
		repo     string
		prNumber int
	}
	byPR := make(map[prKey][]store.Issue)

	for _, issue := range issues {
		// We need to get repo info from the review
		// For now, parse from issue context — we'll need the review's repo
		// Issues don't store repo directly, so we look it up via review_id
		byPR[prKey{}] = append(byPR[prKey{}], issue)
	}

	// Simpler approach: query reviews with pending issues and sync per review
	reviews, err := rs.store.GetReviewsWithPendingIssues(ctx, from, now)
	if err != nil {
		slog.Error("reaction sync: failed to get reviews", "error", err)
		return
	}

	synced := 0
	for _, review := range reviews {
		parts := strings.SplitN(review.Repo, "/", 2)
		if len(parts) != 2 {
			continue
		}
		owner, repo := parts[0], parts[1]

		installID := int64(0)
		if review.InstallationID != nil {
			installID = *review.InstallationID
		}
		if installID == 0 {
			continue
		}

		client, err := rs.ghFactory.Client(installID)
		if err != nil {
			slog.Error("reaction sync: failed to get github client", "install_id", installID, "error", err)
			continue
		}

		// List all review comments on this PR
		comments, _, err := client.PullRequests.ListComments(ctx, owner, repo, review.PRNumber, &gh.PullRequestListCommentsOptions{
			Sort:      "created",
			Direction: "desc",
			Since:     from,
		})
		if err != nil {
			slog.Error("reaction sync: failed to list comments", "repo", review.Repo, "pr", review.PRNumber, "error", err)
			continue
		}

		for _, comment := range comments {
			commentID := comment.GetID()
			reactions := comment.GetReactions()
			if reactions == nil {
				continue
			}

			plusOne := reactions.GetPlusOne()
			minusOne := reactions.GetMinusOne()

			var validation string
			if plusOne > 0 && plusOne > minusOne {
				validation = "confirmed"
			} else if minusOne > 0 {
				validation = "false_positive"
			} else {
				continue
			}

			err := rs.store.ValidateIssueByCommentID(ctx, commentID, validation, "reaction")
			if err == nil {
				synced++
			}
		}
	}

	if synced > 0 {
		slog.Info("reaction sync complete", "synced", synced)
	}
}
