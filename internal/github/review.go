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

// PostReviewResult contains the GitHub comment IDs for each inline comment.
type PostReviewResult struct {
	ReviewID   int64
	CommentIDs []int64 // one per inline comment, in order
}

func PostReview(ctx context.Context, client *gh.Client, owner, repo string, prNumber int, commitSHA string, data *ReviewData) (*PostReviewResult, error) {
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

	review, _, err := client.PullRequests.CreateReview(ctx, owner, repo, prNumber, reviewReq)
	if err != nil {
		return nil, fmt.Errorf("posting review: %w", err)
	}

	result := &PostReviewResult{
		ReviewID: review.GetID(),
	}

	// Fetch the review's comments to get their IDs
	if len(data.Comments) > 0 {
		reviewComments, _, err := client.PullRequests.ListReviewComments(ctx, owner, repo, prNumber, review.GetID(), nil)
		if err == nil {
			for _, rc := range reviewComments {
				result.CommentIDs = append(result.CommentIDs, rc.GetID())
			}
		}
	}

	return result, nil
}

// PRInfo holds metadata about a pull request.
type PRInfo struct {
	HeadSHA  string `json:"head_sha"`
	BaseRef  string `json:"base_ref"`
	Author   string `json:"author"`
	Repo     string `json:"repo,omitempty"`      // optional, used by local fixtures
	PRNumber int    `json:"pr_number,omitempty"` // optional, used by local fixtures
}

func GetPRInfo(ctx context.Context, client *gh.Client, owner, repo string, prNumber int) (*PRInfo, error) {
	pr, _, err := client.PullRequests.Get(ctx, owner, repo, prNumber)
	if err != nil {
		return nil, fmt.Errorf("getting PR: %w", err)
	}
	return &PRInfo{
		HeadSHA: pr.GetHead().GetSHA(),
		BaseRef: pr.GetBase().GetRef(),
		Author:  pr.GetUser().GetLogin(),
	}, nil
}

// GetPRHead is a compatibility wrapper around GetPRInfo.
func GetPRHead(ctx context.Context, client *gh.Client, owner, repo string, prNumber int) (sha string, base string, err error) {
	info, err := GetPRInfo(ctx, client, owner, repo, prNumber)
	if err != nil {
		return "", "", err
	}
	return info.HeadSHA, info.BaseRef, nil
}
