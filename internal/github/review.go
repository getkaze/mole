package github

import (
	"context"
	"fmt"

	gh "github.com/google/go-github/v72/github"
)

type ReviewComment struct {
	File string
	Line int
	Body string
}

type ReviewData struct {
	Body     string
	Comments []ReviewComment
}

func PostReview(ctx context.Context, client *gh.Client, owner, repo string, prNumber int, commitSHA string, data *ReviewData) error {
	comments := make([]*gh.DraftReviewComment, 0, len(data.Comments))
	for _, c := range data.Comments {
		line := c.Line
		comments = append(comments, &gh.DraftReviewComment{
			Path: gh.Ptr(c.File),
			Line: &line,
			Body: gh.Ptr(c.Body),
			Side: gh.Ptr("RIGHT"),
		})
	}

	reviewReq := &gh.PullRequestReviewRequest{
		CommitID: gh.Ptr(commitSHA),
		Body:     gh.Ptr(data.Body),
		Event:    gh.Ptr("COMMENT"),
		Comments: comments,
	}

	_, _, err := client.PullRequests.CreateReview(ctx, owner, repo, prNumber, reviewReq)
	if err != nil {
		return fmt.Errorf("posting review: %w", err)
	}

	return nil
}

func GetPRHead(ctx context.Context, client *gh.Client, owner, repo string, prNumber int) (sha string, base string, err error) {
	pr, _, err := client.PullRequests.Get(ctx, owner, repo, prNumber)
	if err != nil {
		return "", "", fmt.Errorf("getting PR: %w", err)
	}
	return pr.GetHead().GetSHA(), pr.GetBase().GetRef(), nil
}
