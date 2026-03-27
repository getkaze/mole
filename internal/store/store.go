package store

import (
	"context"
	"time"
)

type Store interface {
	// Reviews
	SaveReview(ctx context.Context, r *Review) (int64, error)
	IsIgnored(ctx context.Context, repo string, prNumber int) (bool, error)
	IgnorePR(ctx context.Context, repo string, prNumber int) error

	// Issues
	SaveIssues(ctx context.Context, reviewID int64, issues []Issue) ([]int64, error)
	UpdateIssueCommentID(ctx context.Context, issueID int64, commentID int64) error
	ValidateIssueByCommentID(ctx context.Context, githubCommentID int64, validation string, validatedBy string) error
	GetIssuesByPR(ctx context.Context, repo string, prNumber int) ([]Issue, error)
	GetIssuesByDeveloper(ctx context.Context, developer string, from, to time.Time) ([]Issue, error)
	GetIssuesByModule(ctx context.Context, module string, from, to time.Time) ([]Issue, error)
	GetAcceptanceRate(ctx context.Context, developer string, from, to time.Time) (*AcceptanceRate, error)
	GetOverallAcceptanceRate(ctx context.Context, from, to time.Time) (*AcceptanceRate, error)
	GetPendingValidationIssues(ctx context.Context, from, to time.Time) ([]Issue, error)
	GetReviewsWithPendingIssues(ctx context.Context, from, to time.Time) ([]Review, error)

	// Installations
	UpsertInstallation(ctx context.Context, inst *Installation) error
	AddRepository(ctx context.Context, repo *Repository) error
	RemoveRepository(ctx context.Context, githubRepoID int64) error
	GetInstallation(ctx context.Context, githubInstallID int64) (*Installation, error)

	// Developer Metrics
	UpsertDevMetrics(ctx context.Context, m *DeveloperMetrics) error
	GetDevMetrics(ctx context.Context, developer string, periodType string, from, to time.Time) ([]DeveloperMetrics, error)
	GetDevStreak(ctx context.Context, developer string) (int, error)
	ListAllDevMetrics(ctx context.Context, periodType string, from, to time.Time) ([]DeveloperMetrics, error)

	// Module Metrics
	UpsertModuleMetrics(ctx context.Context, m *ModuleMetrics) error
	GetModuleMetrics(ctx context.Context, module string, periodType string, from, to time.Time) ([]ModuleMetrics, error)
	ListAllModuleMetrics(ctx context.Context, periodType string, from, to time.Time) ([]ModuleMetrics, error)

	// Reviews (aggregated)
	GetAvgScoreByDeveloper(ctx context.Context, developer string, from, to time.Time) (float64, error)

	// Issues (aggregated)
	ListActiveDevelopers(ctx context.Context, from, to time.Time) ([]string, error)
	ListActiveModules(ctx context.Context, from, to time.Time) ([]string, error)
	ListTopIssuePatterns(ctx context.Context, from, to time.Time, limit int) ([]IssuePattern, error)

	// Access Control
	GetAccess(ctx context.Context, githubUser string) (*DashboardAccess, error)
	UpsertAccess(ctx context.Context, access *DashboardAccess) error

	// Infrastructure
	Ping(ctx context.Context) error
	Close() error
}
