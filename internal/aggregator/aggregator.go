package aggregator

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/getkaze/mole/internal/store"
)

// Aggregator periodically computes developer and module metrics from issues.
type Aggregator struct {
	store          store.Store
	interval       time.Duration
	reactionSyncer *ReactionSyncer
}

// New creates an aggregator with the given interval.
func New(s store.Store, interval time.Duration, opts ...Option) *Aggregator {
	if interval <= 0 {
		interval = time.Hour
	}
	a := &Aggregator{store: s, interval: interval}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

// Option configures the aggregator.
type Option func(*Aggregator)

// WithReactionSyncer adds reaction syncing to the aggregation cycle.
func WithReactionSyncer(rs *ReactionSyncer) Option {
	return func(a *Aggregator) {
		a.reactionSyncer = rs
	}
}

// Run starts the aggregation loop. Blocks until ctx is cancelled.
func (a *Aggregator) Run(ctx context.Context) {
	slog.Info("aggregator started", "interval", a.interval)

	// Run once immediately on startup
	a.aggregate(ctx)

	ticker := time.NewTicker(a.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			a.aggregate(ctx)
		case <-ctx.Done():
			slog.Info("aggregator stopped")
			return
		}
	}
}

func (a *Aggregator) aggregate(ctx context.Context) {
	// Sync reactions before aggregating metrics
	if a.reactionSyncer != nil {
		a.reactionSyncer.Sync(ctx)
	}

	now := time.Now()

	// Weekly: last 7 days
	weekStart := now.AddDate(0, 0, -7)
	a.aggregateDevMetrics(ctx, "weekly", weekStart, now)
	a.aggregateModuleMetrics(ctx, "weekly", weekStart, now)

	// Monthly: last 30 days
	monthStart := now.AddDate(0, 0, -30)
	a.aggregateDevMetrics(ctx, "monthly", monthStart, now)
	a.aggregateModuleMetrics(ctx, "monthly", monthStart, now)

	slog.Debug("aggregation cycle complete")
}

func (a *Aggregator) aggregateDevMetrics(ctx context.Context, periodType string, from, to time.Time) {
	// Get all developers with issues in this period
	developers, err := a.getActiveDevelopers(ctx, from, to)
	if err != nil {
		slog.Error("failed to get active developers", "error", err)
		return
	}

	for _, dev := range developers {
		issues, err := a.store.GetIssuesByDeveloper(ctx, dev, from, to)
		if err != nil {
			slog.Error("failed to get issues for developer", "developer", dev, "error", err)
			continue
		}

		byCat := countByField(issues, func(i store.Issue) string { return i.Category })
		bySev := countByField(issues, func(i store.Issue) string { return i.Severity })
		byCatJSON, _ := json.Marshal(byCat)
		bySevJSON, _ := json.Marshal(bySev)

		streak := a.calculateStreak(issues)
		badges := evaluateBadges(countUniqueReviews(issues), streak, byCat)
		badgesJSON, _ := json.Marshal(badges)

		avgScore, err := a.store.GetAvgScoreByDeveloper(ctx, dev, from, to)
		if err != nil {
			slog.Error("failed to get avg score", "developer", dev, "error", err)
		}

		m := &store.DeveloperMetrics{
			Developer:        dev,
			PeriodType:       periodType,
			PeriodStart:      from,
			PeriodEnd:        to,
			TotalReviews:     countUniqueReviews(issues),
			AvgScore:         avgScore,
			IssuesByCategory: string(byCatJSON),
			IssuesBySeverity: string(bySevJSON),
			StreakCleanPRs:   streak,
			Badges:           string(badgesJSON),
		}

		if err := a.store.UpsertDevMetrics(ctx, m); err != nil {
			slog.Error("failed to upsert dev metrics", "developer", dev, "error", err)
		}
	}
}

func (a *Aggregator) aggregateModuleMetrics(ctx context.Context, periodType string, from, to time.Time) {
	modules, err := a.getActiveModules(ctx, from, to)
	if err != nil {
		slog.Error("failed to get active modules", "error", err)
		return
	}

	for _, mod := range modules {
		issues, err := a.store.GetIssuesByModule(ctx, mod, from, to)
		if err != nil {
			slog.Error("failed to get issues for module", "module", mod, "error", err)
			continue
		}

		criticalCount := 0
		for _, i := range issues {
			if i.Severity == "critical" {
				criticalCount++
			}
		}

		// Health score: 100 - (critical * 10 + total * 1), floor at 0
		health := 100.0 - float64(criticalCount*10) - float64(len(issues))
		if health < 0 {
			health = 0
		}

		m := &store.ModuleMetrics{
			ModuleName:  mod,
			PeriodType:  periodType,
			PeriodStart: from,
			PeriodEnd:   to,
			AvgScore:    0,
			HealthScore: health,
			TotalIssues: len(issues),
			DebtItems:   criticalCount,
		}

		if err := a.store.UpsertModuleMetrics(ctx, m); err != nil {
			slog.Error("failed to upsert module metrics", "module", mod, "error", err)
		}
	}
}

func (a *Aggregator) getActiveDevelopers(ctx context.Context, from, to time.Time) ([]string, error) {
	return a.store.ListActiveDevelopers(ctx, from, to)
}

func (a *Aggregator) getActiveModules(ctx context.Context, from, to time.Time) ([]string, error) {
	return a.store.ListActiveModules(ctx, from, to)
}

func countByField(issues []store.Issue, fn func(store.Issue) string) map[string]int {
	counts := make(map[string]int)
	for _, i := range issues {
		counts[fn(i)]++
	}
	return counts
}

func countUniqueReviews(issues []store.Issue) int {
	seen := make(map[int64]bool)
	for _, i := range issues {
		seen[i.ReviewID] = true
	}
	return len(seen)
}

// calculateStreak counts consecutive reviews (by review_id) with zero critical issues.
func (a *Aggregator) calculateStreak(issues []store.Issue) int {
	if len(issues) == 0 {
		return 0
	}

	// Group issues by review_id
	byReview := make(map[int64]bool) // true = has critical
	var reviewOrder []int64
	seen := make(map[int64]bool)
	for _, i := range issues {
		if !seen[i.ReviewID] {
			seen[i.ReviewID] = true
			reviewOrder = append(reviewOrder, i.ReviewID)
		}
		if i.Severity == "critical" {
			byReview[i.ReviewID] = true
		}
	}

	// Count from most recent backwards
	streak := 0
	for j := len(reviewOrder) - 1; j >= 0; j-- {
		if byReview[reviewOrder[j]] {
			break
		}
		streak++
	}
	return streak
}
