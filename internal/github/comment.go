package github

import (
	"context"
	"fmt"

	gh "github.com/google/go-github/v72/github"
)

// PostComment creates a simple issue comment on a PR (not a review comment).
// Returns the comment ID for later editing.
func PostComment(ctx context.Context, client *gh.Client, owner, repo string, pr int, body string) (int64, error) {
	comment, _, err := client.Issues.CreateComment(ctx, owner, repo, pr, &gh.IssueComment{
		Body: gh.Ptr(body),
	})
	if err != nil {
		return 0, fmt.Errorf("posting comment: %w", err)
	}
	return comment.GetID(), nil
}

// EditComment updates an existing issue comment.
func EditComment(ctx context.Context, client *gh.Client, owner, repo string, commentID int64, body string) error {
	_, _, err := client.Issues.EditComment(ctx, owner, repo, commentID, &gh.IssueComment{
		Body: gh.Ptr(body),
	})
	if err != nil {
		return fmt.Errorf("editing comment: %w", err)
	}
	return nil
}
