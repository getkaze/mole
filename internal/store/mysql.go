package store

import (
	"context"
	"database/sql"
	"fmt"

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

func (s *MySQLStore) SaveReview(ctx context.Context, r *Review) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO reviews (repo, pr_number, review_type, model, input_tokens, output_tokens, status, summary, error_message)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.Repo, r.PRNumber, r.ReviewType, r.Model,
		r.InputTokens, r.OutputTokens, r.Status, r.Summary, r.ErrorMessage,
	)
	return err
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

func (s *MySQLStore) IgnorePR(ctx context.Context, repo string, prNumber int) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT IGNORE INTO ignored_prs (repo, pr_number) VALUES (?, ?)`,
		repo, prNumber,
	)
	return err
}
