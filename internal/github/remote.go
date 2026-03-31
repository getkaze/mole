package github

import (
	"context"
	"strings"
)

// RemoteGateway implements Gateway using the real GitHub API.
type RemoteGateway struct {
	factory   *ClientFactory
	installID int64
}

// NewRemoteGatewayFactory returns a GatewayFactory that creates RemoteGateway
// instances backed by the given ClientFactory.
func NewRemoteGatewayFactory(cf *ClientFactory) GatewayFactory {
	return func(installID int64) Gateway {
		return &RemoteGateway{factory: cf, installID: installID}
	}
}

func (r *RemoteGateway) split(repo string) (owner, name string) {
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return repo, ""
}

func (r *RemoteGateway) GetPRInfo(ctx context.Context, repo string, pr int) (*PRInfo, error) {
	client, err := r.factory.Client(r.installID)
	if err != nil {
		return nil, err
	}
	owner, name := r.split(repo)
	return GetPRInfo(ctx, client, owner, name, pr)
}

func (r *RemoteGateway) FetchDiff(ctx context.Context, repo string, pr int) ([]FileDiff, error) {
	client, err := r.factory.Client(r.installID)
	if err != nil {
		return nil, err
	}
	owner, name := r.split(repo)
	return FetchDiff(ctx, client, owner, name, pr)
}

func (r *RemoteGateway) LoadContext(ctx context.Context, repo, ref string) (*ContextResult, error) {
	client, err := r.factory.Client(r.installID)
	if err != nil {
		return nil, err
	}
	owner, name := r.split(repo)
	return LoadContext(ctx, client, owner, name, ref)
}

func (r *RemoteGateway) LoadRepoConfig(ctx context.Context, repo, ref string) (*RepoConfig, error) {
	client, err := r.factory.Client(r.installID)
	if err != nil {
		return nil, err
	}
	owner, name := r.split(repo)
	return LoadRepoConfig(ctx, client, owner, name, ref)
}

func (r *RemoteGateway) PostReview(ctx context.Context, repo string, pr int, sha string, data *ReviewData) (*PostReviewResult, error) {
	client, err := r.factory.Client(r.installID)
	if err != nil {
		return nil, err
	}
	owner, name := r.split(repo)
	return PostReview(ctx, client, owner, name, pr, sha, data)
}

func (r *RemoteGateway) AddReaction(ctx context.Context, repo string, pr int, commentID int64, reaction string) {
	client, err := r.factory.Client(r.installID)
	if err != nil {
		return
	}
	owner, name := r.split(repo)
	AddReaction(ctx, client, owner, name, pr, commentID, reaction)
}
