package store

import "context"

type Store interface {
	SaveReview(ctx context.Context, r *Review) error
	IsIgnored(ctx context.Context, repo string, prNumber int) (bool, error)
	IgnorePR(ctx context.Context, repo string, prNumber int) error
	Ping(ctx context.Context) error
	Close() error
}
