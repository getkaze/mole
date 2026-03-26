package github

import (
	"context"
	"log/slog"

	gh "github.com/google/go-github/v72/github"
)

// AddReaction adds a reaction to the triggering event.
// If commentID > 0, reacts to the comment; otherwise reacts to the PR itself.
func AddReaction(ctx context.Context, client *gh.Client, owner, repo string, prNumber int, commentID int64, reaction string) {
	var err error
	if commentID > 0 {
		_, _, err = client.Reactions.CreateIssueCommentReaction(ctx, owner, repo, commentID, reaction)
	} else {
		_, _, err = client.Reactions.CreateIssueReaction(ctx, owner, repo, prNumber, reaction)
	}
	if err != nil {
		slog.Warn("failed to add reaction", "reaction", reaction, "repo", owner+"/"+repo, "pr", prNumber, "error", err)
	}
}
