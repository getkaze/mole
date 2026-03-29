package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type MySQLStore struct {
	db *sql.DB
}

func NewMySQL(dsn string) (*MySQLStore, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening mysql: %w", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("pinging mysql: %w", err)
	}

	return &MySQLStore{db: db}, nil
}

func (s *MySQLStore) DB() *sql.DB {
	return s.db
}

func (s *MySQLStore) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *MySQLStore) Close() error {
	return s.db.Close()
}

// Reviews

func (s *MySQLStore) SaveReview(ctx context.Context, r *Review) (int64, error) {
	result, err := s.db.ExecContext(ctx,
		`INSERT INTO reviews (repo, pr_number, pr_author, review_type, model, score, input_tokens, output_tokens, status, summary, error_message, installation_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.Repo, r.PRNumber, r.PRAuthor, r.ReviewType, r.Model, r.Score,
		r.InputTokens, r.OutputTokens, r.Status, r.Summary, r.ErrorMessage, r.InstallationID,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (s *MySQLStore) IsIgnored(ctx context.Context, repo string, prNumber int) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM ignored_prs WHERE repo = ? AND pr_number = ?`,
		repo, prNumber,
	).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *MySQLStore) GetAvgScoreByDeveloper(ctx context.Context, developer string, from, to time.Time) (float64, error) {
	var avg sql.NullFloat64
	err := s.db.QueryRowContext(ctx,
		`SELECT AVG(score) FROM reviews WHERE pr_author = ? AND score IS NOT NULL AND created_at BETWEEN ? AND ?`,
		developer, from, to,
	).Scan(&avg)
	if err != nil {
		return 0, err
	}
	if !avg.Valid {
		return 0, nil
	}
	return avg.Float64, nil
}

func (s *MySQLStore) IgnorePR(ctx context.Context, repo string, prNumber int) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT IGNORE INTO ignored_prs (repo, pr_number) VALUES (?, ?)`,
		repo, prNumber,
	)
	return err
}

// Issues

func (s *MySQLStore) SaveIssues(ctx context.Context, reviewID int64, issues []Issue) ([]int64, error) {
	if len(issues) == 0 {
		return nil, nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO issues (review_id, pr_author, category, subcategory, severity, file_path, line_number, description, suggestion, module_name)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
	)
	if err != nil {
		return nil, fmt.Errorf("prepare: %w", err)
	}
	defer stmt.Close()

	var ids []int64
	for _, issue := range issues {
		result, err := stmt.ExecContext(ctx,
			reviewID, issue.PRAuthor, issue.Category, issue.Subcategory, issue.Severity,
			issue.FilePath, issue.LineNumber, issue.Description, issue.Suggestion, issue.ModuleName,
		)
		if err != nil {
			return nil, fmt.Errorf("insert issue: %w", err)
		}
		id, _ := result.LastInsertId()
		ids = append(ids, id)
	}

	return ids, tx.Commit()
}

func (s *MySQLStore) UpdateIssueCommentID(ctx context.Context, issueID int64, commentID int64) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE issues SET github_comment_id = ? WHERE id = ?`,
		commentID, issueID,
	)
	return err
}

func (s *MySQLStore) ValidateIssueByCommentID(ctx context.Context, githubCommentID int64, validation string, validatedBy string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE issues SET validation = ?, validated_by = ?, validated_at = NOW() WHERE github_comment_id = ?`,
		validation, validatedBy, githubCommentID,
	)
	return err
}

func (s *MySQLStore) GetIssuesByPR(ctx context.Context, repo string, prNumber int) ([]Issue, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT i.id, i.review_id, i.pr_author, i.category, i.subcategory, i.severity, i.file_path, i.line_number, i.description, i.suggestion, i.module_name, i.validation, i.created_at
		 FROM issues i
		 JOIN reviews r ON r.id = i.review_id
		 WHERE r.repo = ? AND r.pr_number = ?
		 ORDER BY i.created_at`,
		repo, prNumber,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanIssues(rows)
}

func (s *MySQLStore) GetAcceptanceRate(ctx context.Context, developer string, from, to time.Time) (*AcceptanceRate, error) {
	var total, confirmed, falsePositive, pending int
	err := s.db.QueryRowContext(ctx,
		`SELECT
			COUNT(*),
			COALESCE(SUM(CASE WHEN validation = 'confirmed' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN validation = 'false_positive' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN validation = 'pending' THEN 1 ELSE 0 END), 0)
		 FROM issues WHERE pr_author = ? AND created_at BETWEEN ? AND ?`,
		developer, from, to,
	).Scan(&total, &confirmed, &falsePositive, &pending)
	if err != nil {
		return nil, err
	}

	rate := 0.0
	if confirmed+falsePositive > 0 {
		rate = float64(confirmed) / float64(confirmed+falsePositive) * 100
	}

	return &AcceptanceRate{
		Total:         total,
		Confirmed:     confirmed,
		FalsePositive: falsePositive,
		Pending:       pending,
		Rate:          rate,
	}, nil
}

func (s *MySQLStore) GetOverallAcceptanceRate(ctx context.Context, from, to time.Time) (*AcceptanceRate, error) {
	var total, confirmed, falsePositive, pending int
	err := s.db.QueryRowContext(ctx,
		`SELECT
			COUNT(*),
			COALESCE(SUM(CASE WHEN validation = 'confirmed' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN validation = 'false_positive' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN validation = 'pending' THEN 1 ELSE 0 END), 0)
		 FROM issues WHERE created_at BETWEEN ? AND ?`,
		from, to,
	).Scan(&total, &confirmed, &falsePositive, &pending)
	if err != nil {
		return nil, err
	}

	rate := 0.0
	if confirmed+falsePositive > 0 {
		rate = float64(confirmed) / float64(confirmed+falsePositive) * 100
	}

	return &AcceptanceRate{
		Total:         total,
		Confirmed:     confirmed,
		FalsePositive: falsePositive,
		Pending:       pending,
		Rate:          rate,
	}, nil
}

func (s *MySQLStore) GetIssuesByDeveloper(ctx context.Context, developer string, from, to time.Time) ([]Issue, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, review_id, pr_author, category, subcategory, severity, file_path, line_number, description, suggestion, module_name, validation, created_at
		 FROM issues WHERE pr_author = ? AND created_at BETWEEN ? AND ? ORDER BY created_at`,
		developer, from, to,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanIssues(rows)
}

func (s *MySQLStore) GetPendingValidationIssues(ctx context.Context, from, to time.Time) ([]Issue, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, review_id, pr_author, category, subcategory, severity, file_path, line_number, description, suggestion, module_name, validation, created_at
		 FROM issues WHERE validation = 'pending' AND github_comment_id IS NOT NULL AND created_at BETWEEN ? AND ? ORDER BY created_at`,
		from, to,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanIssues(rows)
}

func (s *MySQLStore) GetReviewsWithPendingIssues(ctx context.Context, from, to time.Time) ([]Review, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT DISTINCT r.id, r.repo, r.pr_number, r.pr_author, r.review_type, r.model, r.score, r.input_tokens, r.output_tokens, r.status, r.summary, r.error_message, r.installation_id, r.created_at
		 FROM reviews r
		 JOIN issues i ON i.review_id = r.id
		 WHERE i.validation = 'pending' AND i.github_comment_id IS NOT NULL AND r.created_at BETWEEN ? AND ?`,
		from, to,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reviews []Review
	for rows.Next() {
		var r Review
		var score sql.NullInt64
		var installID sql.NullInt64
		var summary, errMsg sql.NullString
		if err := rows.Scan(
			&r.ID, &r.Repo, &r.PRNumber, &r.PRAuthor, &r.ReviewType, &r.Model,
			&score, &r.InputTokens, &r.OutputTokens, &r.Status, &summary, &errMsg, &installID, &r.CreatedAt,
		); err != nil {
			return nil, err
		}
		if score.Valid {
			s := int(score.Int64)
			r.Score = &s
		}
		if installID.Valid {
			r.InstallationID = &installID.Int64
		}
		r.Summary = summary.String
		r.ErrorMessage = errMsg.String
		reviews = append(reviews, r)
	}
	return reviews, rows.Err()
}

func (s *MySQLStore) GetIssuesByModule(ctx context.Context, repo, module string, from, to time.Time) ([]Issue, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT i.id, i.review_id, i.pr_author, i.category, i.subcategory, i.severity, i.file_path, i.line_number, i.description, i.suggestion, i.module_name, i.validation, i.created_at
		 FROM issues i
		 JOIN reviews r ON r.id = i.review_id
		 WHERE r.repo = ? AND i.module_name = ? AND i.created_at BETWEEN ? AND ? ORDER BY i.created_at`,
		repo, module, from, to,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanIssues(rows)
}

func scanIssues(rows *sql.Rows) ([]Issue, error) {
	var issues []Issue
	for rows.Next() {
		var i Issue
		var suggestion, module, validation sql.NullString
		if err := rows.Scan(
			&i.ID, &i.ReviewID, &i.PRAuthor, &i.Category, &i.Subcategory, &i.Severity,
			&i.FilePath, &i.LineNumber, &i.Description, &suggestion, &module, &validation, &i.CreatedAt,
		); err != nil {
			return nil, err
		}
		i.Suggestion = suggestion.String
		i.ModuleName = module.String
		i.Validation = validation.String
		issues = append(issues, i)
	}
	return issues, rows.Err()
}

// Installations

func (s *MySQLStore) UpsertInstallation(ctx context.Context, inst *Installation) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO installations (github_installation_id, owner, status)
		 VALUES (?, ?, ?)
		 ON DUPLICATE KEY UPDATE owner = VALUES(owner), status = VALUES(status)`,
		inst.GitHubInstallationID, inst.Owner, inst.Status,
	)
	return err
}

func (s *MySQLStore) AddRepository(ctx context.Context, repo *Repository) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO repositories (installation_id, github_repo_id, full_name, active)
		 VALUES (?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE full_name = VALUES(full_name), active = VALUES(active)`,
		repo.InstallationID, repo.GitHubRepoID, repo.FullName, repo.Active,
	)
	return err
}

func (s *MySQLStore) RemoveRepository(ctx context.Context, githubRepoID int64) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE repositories SET active = FALSE WHERE github_repo_id = ?`,
		githubRepoID,
	)
	return err
}

func (s *MySQLStore) GetInstallation(ctx context.Context, githubInstallID int64) (*Installation, error) {
	var inst Installation
	err := s.db.QueryRowContext(ctx,
		`SELECT id, github_installation_id, owner, status, created_at, updated_at
		 FROM installations WHERE github_installation_id = ?`,
		githubInstallID,
	).Scan(&inst.ID, &inst.GitHubInstallationID, &inst.Owner, &inst.Status, &inst.CreatedAt, &inst.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &inst, nil
}

// Developer Metrics

func (s *MySQLStore) UpsertDevMetrics(ctx context.Context, m *DeveloperMetrics) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO developer_metrics (developer, period_type, period_start, period_end, total_reviews, avg_score, issues_by_category, issues_by_severity, streak_clean_prs, badges)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE
		   total_reviews = VALUES(total_reviews),
		   avg_score = VALUES(avg_score),
		   issues_by_category = VALUES(issues_by_category),
		   issues_by_severity = VALUES(issues_by_severity),
		   streak_clean_prs = VALUES(streak_clean_prs),
		   badges = VALUES(badges)`,
		m.Developer, m.PeriodType, m.PeriodStart, m.PeriodEnd,
		m.TotalReviews, m.AvgScore, m.IssuesByCategory, m.IssuesBySeverity,
		m.StreakCleanPRs, m.Badges,
	)
	return err
}

func (s *MySQLStore) GetDevMetrics(ctx context.Context, developer string, periodType string, from, to time.Time) ([]DeveloperMetrics, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, developer, period_type, period_start, period_end, total_reviews, avg_score,
		        issues_by_category, issues_by_severity, streak_clean_prs, badges, created_at
		 FROM developer_metrics
		 WHERE developer = ? AND period_type = ? AND period_start BETWEEN ? AND ?
		 ORDER BY period_start`,
		developer, periodType, from, to,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []DeveloperMetrics
	for rows.Next() {
		var m DeveloperMetrics
		var byCat, bySev, badges sql.NullString
		if err := rows.Scan(
			&m.ID, &m.Developer, &m.PeriodType, &m.PeriodStart, &m.PeriodEnd,
			&m.TotalReviews, &m.AvgScore, &byCat, &bySev, &m.StreakCleanPRs, &badges, &m.CreatedAt,
		); err != nil {
			return nil, err
		}
		m.IssuesByCategory = byCat.String
		m.IssuesBySeverity = bySev.String
		m.Badges = badges.String
		metrics = append(metrics, m)
	}
	return metrics, rows.Err()
}

func (s *MySQLStore) GetDevStreak(ctx context.Context, developer string) (int, error) {
	// Get the most recent streak_clean_prs value
	var streak int
	err := s.db.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(streak_clean_prs), 0)
		 FROM developer_metrics
		 WHERE developer = ?`,
		developer,
	).Scan(&streak)
	if err != nil {
		return 0, err
	}
	return streak, nil
}

// Module Metrics

func (s *MySQLStore) UpsertModuleMetrics(ctx context.Context, m *ModuleMetrics) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO module_metrics (repo, module_name, period_type, period_start, period_end, avg_score, health_score, total_issues, debt_items)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE
		   avg_score = VALUES(avg_score),
		   health_score = VALUES(health_score),
		   total_issues = VALUES(total_issues),
		   debt_items = VALUES(debt_items)`,
		m.Repo, m.ModuleName, m.PeriodType, m.PeriodStart, m.PeriodEnd,
		m.AvgScore, m.HealthScore, m.TotalIssues, m.DebtItems,
	)
	return err
}

func (s *MySQLStore) GetModuleMetrics(ctx context.Context, repo, module string, periodType string, from, to time.Time) ([]ModuleMetrics, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, repo, module_name, period_type, period_start, period_end, avg_score, health_score, total_issues, debt_items, created_at
		 FROM module_metrics
		 WHERE repo = ? AND module_name = ? AND period_type = ? AND period_start BETWEEN ? AND ?
		 ORDER BY period_start`,
		repo, module, periodType, from, to,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []ModuleMetrics
	for rows.Next() {
		var m ModuleMetrics
		if err := rows.Scan(
			&m.ID, &m.Repo, &m.ModuleName, &m.PeriodType, &m.PeriodStart, &m.PeriodEnd,
			&m.AvgScore, &m.HealthScore, &m.TotalIssues, &m.DebtItems, &m.CreatedAt,
		); err != nil {
			return nil, err
		}
		metrics = append(metrics, m)
	}
	return metrics, rows.Err()
}

// Access Control

func (s *MySQLStore) GetAccess(ctx context.Context, githubUser string) (*DashboardAccess, error) {
	var a DashboardAccess
	err := s.db.QueryRowContext(ctx,
		`SELECT id, github_user, role, individual_visibility, created_at
		 FROM dashboard_access WHERE github_user = ?`,
		githubUser,
	).Scan(&a.ID, &a.GitHubUser, &a.Role, &a.IndividualVisibility, &a.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (s *MySQLStore) UpsertAccess(ctx context.Context, access *DashboardAccess) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO dashboard_access (github_user, role, individual_visibility)
		 VALUES (?, ?, ?)
		 ON DUPLICATE KEY UPDATE role = VALUES(role), individual_visibility = VALUES(individual_visibility)`,
		access.GitHubUser, access.Role, access.IndividualVisibility,
	)
	return err
}

// Score recalculation

func (s *MySQLStore) GetReviewIDsWithFalsePositives(ctx context.Context, from, to time.Time) ([]int64, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT DISTINCT review_id FROM issues
		 WHERE validation = 'false_positive' AND created_at BETWEEN ? AND ?`,
		from, to,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (s *MySQLStore) GetNonFalsePositiveSeverities(ctx context.Context, reviewID int64) ([]string, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT severity FROM issues
		 WHERE review_id = ? AND validation != 'false_positive'`,
		reviewID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var severities []string
	for rows.Next() {
		var sev string
		if err := rows.Scan(&sev); err != nil {
			return nil, err
		}
		severities = append(severities, sev)
	}
	return severities, rows.Err()
}

func (s *MySQLStore) UpdateReviewScore(ctx context.Context, reviewID int64, score int) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE reviews SET score = ? WHERE id = ?`,
		score, reviewID,
	)
	return err
}

// Costs

func (s *MySQLStore) GetTokenUsageSummary(ctx context.Context, from, to time.Time, pricing map[string][2]float64) ([]TokenUsageSummary, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT model, COUNT(*) as reviews, SUM(input_tokens) as input_tokens, SUM(output_tokens) as output_tokens
		 FROM reviews
		 WHERE status = 'success' AND created_at BETWEEN ? AND ?
		 GROUP BY model
		 ORDER BY SUM(input_tokens) + SUM(output_tokens) DESC`,
		from, to,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []TokenUsageSummary
	for rows.Next() {
		var s TokenUsageSummary
		if err := rows.Scan(&s.Model, &s.Reviews, &s.InputTokens, &s.OutputTokens); err != nil {
			return nil, err
		}
		rates, ok := pricing[s.Model]
		if !ok {
			rates = [2]float64{3.00, 15.00} // default to sonnet pricing
		}
		s.InputCost = float64(s.InputTokens) / 1_000_000 * rates[0]
		s.OutputCost = float64(s.OutputTokens) / 1_000_000 * rates[1]
		s.TotalCost = s.InputCost + s.OutputCost
		summaries = append(summaries, s)
	}
	return summaries, rows.Err()
}

func (s *MySQLStore) GetUniquePRCount(ctx context.Context, from, to time.Time) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(DISTINCT CONCAT(repo, '#', pr_number))
		 FROM reviews
		 WHERE status = 'success' AND created_at BETWEEN ? AND ?`,
		from, to,
	).Scan(&count)
	return count, err
}

// List methods for team/module dashboards

func (s *MySQLStore) ListAllDevMetrics(ctx context.Context, periodType string, from, to time.Time) ([]DeveloperMetrics, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, developer, period_type, period_start, period_end, total_reviews, avg_score,
		        issues_by_category, issues_by_severity, streak_clean_prs, badges, created_at
		 FROM developer_metrics
		 WHERE period_type = ? AND period_start BETWEEN ? AND ?
		 ORDER BY developer, period_start`,
		periodType, from, to,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []DeveloperMetrics
	for rows.Next() {
		var m DeveloperMetrics
		var byCat, bySev, badges sql.NullString
		if err := rows.Scan(
			&m.ID, &m.Developer, &m.PeriodType, &m.PeriodStart, &m.PeriodEnd,
			&m.TotalReviews, &m.AvgScore, &byCat, &bySev, &m.StreakCleanPRs, &badges, &m.CreatedAt,
		); err != nil {
			return nil, err
		}
		m.IssuesByCategory = byCat.String
		m.IssuesBySeverity = bySev.String
		m.Badges = badges.String
		metrics = append(metrics, m)
	}
	return metrics, rows.Err()
}

func (s *MySQLStore) ListAllModuleMetrics(ctx context.Context, periodType string, from, to time.Time) ([]ModuleMetrics, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, repo, module_name, period_type, period_start, period_end, avg_score, health_score, total_issues, debt_items, created_at
		 FROM module_metrics
		 WHERE period_type = ? AND period_start BETWEEN ? AND ?
		 ORDER BY repo, module_name`,
		periodType, from, to,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []ModuleMetrics
	for rows.Next() {
		var m ModuleMetrics
		if err := rows.Scan(
			&m.ID, &m.Repo, &m.ModuleName, &m.PeriodType, &m.PeriodStart, &m.PeriodEnd,
			&m.AvgScore, &m.HealthScore, &m.TotalIssues, &m.DebtItems, &m.CreatedAt,
		); err != nil {
			return nil, err
		}
		metrics = append(metrics, m)
	}
	return metrics, rows.Err()
}

func (s *MySQLStore) ListActiveDevelopers(ctx context.Context, from, to time.Time) ([]string, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT DISTINCT pr_author FROM issues WHERE created_at BETWEEN ? AND ? ORDER BY pr_author`,
		from, to,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devs []string
	for rows.Next() {
		var dev string
		if err := rows.Scan(&dev); err != nil {
			return nil, err
		}
		devs = append(devs, dev)
	}
	return devs, rows.Err()
}

func (s *MySQLStore) ListActiveModules(ctx context.Context, from, to time.Time) ([]RepoModule, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT DISTINCT r.repo, i.module_name
		 FROM issues i
		 JOIN reviews r ON r.id = i.review_id
		 WHERE i.module_name IS NOT NULL AND i.module_name != '' AND i.created_at BETWEEN ? AND ?
		 ORDER BY r.repo, i.module_name`,
		from, to,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mods []RepoModule
	for rows.Next() {
		var m RepoModule
		if err := rows.Scan(&m.Repo, &m.ModuleName); err != nil {
			return nil, err
		}
		mods = append(mods, m)
	}
	return mods, rows.Err()
}

func (s *MySQLStore) ListTopIssuePatterns(ctx context.Context, from, to time.Time, limit int) ([]IssuePattern, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT category, subcategory, COUNT(*) as cnt
		 FROM issues
		 WHERE created_at BETWEEN ? AND ?
		   AND validation != 'false_positive'
		 GROUP BY category, subcategory
		 ORDER BY cnt DESC
		 LIMIT ?`,
		from, to, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var patterns []IssuePattern
	for rows.Next() {
		var p IssuePattern
		if err := rows.Scan(&p.Category, &p.Subcategory, &p.Count); err != nil {
			return nil, err
		}
		patterns = append(patterns, p)
	}
	return patterns, rows.Err()
}
