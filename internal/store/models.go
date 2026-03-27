package store

import "time"

type Review struct {
	ID             int64
	Repo           string
	PRNumber       int
	PRAuthor       string
	ReviewType     string // "standard" or "deep"
	Model          string
	Score          *int   // nullable
	InputTokens    int
	OutputTokens   int
	Status         string // "success" or "failed"
	Summary        string
	ErrorMessage   string
	InstallationID *int64 // nullable
	CreatedAt      time.Time
}

type IgnoredPR struct {
	ID        int64
	Repo      string
	PRNumber  int
	CreatedAt time.Time
}

type Issue struct {
	ID              int64
	ReviewID        int64
	PRAuthor        string
	Category        string
	Subcategory     string
	Severity        string // critical, attention, suggestion
	FilePath        string
	LineNumber      int
	Description     string
	Suggestion      string
	ModuleName      string
	GitHubCommentID *int64 // maps to the inline comment posted on GitHub
	Validation      string // pending, confirmed, false_positive
	ValidatedBy     string
	ValidatedAt     *time.Time
	CreatedAt       time.Time
}

type AcceptanceRate struct {
	Total         int
	Confirmed     int
	FalsePositive int
	Pending       int
	Rate          float64 // confirmed / (confirmed + false_positive) * 100
}

type Installation struct {
	ID                   int64
	GitHubInstallationID int64
	Owner                string
	Status               string // active, suspended, removed
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type Repository struct {
	ID             int64
	InstallationID int64
	GitHubRepoID   int64
	FullName       string
	Active         bool
	CreatedAt      time.Time
}

type DeveloperMetrics struct {
	ID               int64
	Developer        string
	PeriodType       string // weekly, monthly
	PeriodStart      time.Time
	PeriodEnd        time.Time
	TotalReviews     int
	AvgScore         float64
	IssuesByCategory string // JSON
	IssuesBySeverity string // JSON
	StreakCleanPRs   int
	Badges           string // JSON
	CreatedAt        time.Time
}

type ModuleMetrics struct {
	ID          int64
	ModuleName  string
	PeriodType  string // weekly, monthly
	PeriodStart time.Time
	PeriodEnd   time.Time
	AvgScore    float64
	HealthScore float64
	TotalIssues int
	DebtItems   int
	CreatedAt   time.Time
}

type IssuePattern struct {
	Category    string
	Subcategory string
	Count       int
}

type TokenUsageSummary struct {
	Model        string
	Reviews      int
	InputTokens  int64
	OutputTokens int64
	InputCost    float64
	OutputCost   float64
	TotalCost    float64
}

type DashboardAccess struct {
	ID                   int64
	GitHubUser           string
	Role                 string // dev, tech_lead, architect, manager
	IndividualVisibility bool
	CreatedAt            time.Time
}
