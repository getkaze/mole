package github

import "context"

// Gateway abstracts all GitHub operations needed by the review pipeline.
// RemoteGateway calls the real GitHub API; LocalGateway reads from fixtures.
type Gateway interface {
	GetPRInfo(ctx context.Context, repo string, pr int) (*PRInfo, error)
	FetchDiff(ctx context.Context, repo string, pr int) ([]FileDiff, error)
	LoadContext(ctx context.Context, repo, ref string) (*ContextResult, error)
	LoadRepoConfig(ctx context.Context, repo, ref string) (*RepoConfig, error)
	PostReview(ctx context.Context, repo string, pr int, sha string, data *ReviewData) (*PostReviewResult, error)
	AddReaction(ctx context.Context, repo string, pr int, commentID int64, reaction string)
	PostComment(ctx context.Context, repo string, pr int, body string) (int64, error)
	EditComment(ctx context.Context, repo string, pr int, commentID int64, body string) error
}

// GatewayFactory creates a Gateway for a given GitHub App installation.
// Remote implementations use installID to create per-installation clients.
// Local implementations ignore installID and return a fixed LocalGateway.
type GatewayFactory func(installID int64) Gateway
