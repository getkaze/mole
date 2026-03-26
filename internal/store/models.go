package store

import "time"

type Review struct {
	ID           int64
	Repo         string
	PRNumber     int
	ReviewType   string // "standard" or "deep"
	Model        string
	InputTokens  int
	OutputTokens int
	Status       string // "success" or "failed"
	Summary      string
	ErrorMessage string
	CreatedAt    time.Time
}

type IgnoredPR struct {
	ID        int64
	Repo      string
	PRNumber  int
	CreatedAt time.Time
}
