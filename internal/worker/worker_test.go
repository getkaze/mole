package worker

import (
	"context"
	"errors"
	"testing"

	"github.com/getkaze/mole/internal/queue"
)

func TestNewPool(t *testing.T) {
	p := NewPool(nil, nil, 5)
	if p.count != 5 {
		t.Errorf("count = %d, want 5", p.count)
	}
}

func TestProcessWithRetry_SuccessFirstAttempt(t *testing.T) {
	var calls int
	fn := func(ctx context.Context, job queue.Job) error {
		calls++
		return nil
	}

	p := &Pool{reviewFn: fn}
	job := &queue.Job{ID: "test-1", Repo: "owner/repo", PRNumber: 1}
	p.processWithRetry(context.Background(), 0, job)

	if calls != 1 {
		t.Errorf("calls = %d, want 1", calls)
	}
	if job.Attempts != 1 {
		t.Errorf("attempts = %d, want 1", job.Attempts)
	}
}

func TestProcessWithRetry_SuccessAfterRetry(t *testing.T) {
	var calls int
	fn := func(ctx context.Context, job queue.Job) error {
		calls++
		if calls < 2 {
			return errors.New("transient error")
		}
		return nil
	}

	p := &Pool{reviewFn: fn}
	job := &queue.Job{ID: "test-2", Repo: "owner/repo", PRNumber: 1}
	p.processWithRetry(context.Background(), 0, job)

	if calls != 2 {
		t.Errorf("calls = %d, want 2", calls)
	}
	if job.Attempts != 2 {
		t.Errorf("attempts = %d, want 2", job.Attempts)
	}
}

func TestProcessWithRetry_CancelledContext(t *testing.T) {
	var calls int
	fn := func(ctx context.Context, job queue.Job) error {
		calls++
		return errors.New("fail")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	p := &Pool{reviewFn: fn}
	job := &queue.Job{ID: "test-3", Repo: "owner/repo", PRNumber: 1}
	p.processWithRetry(ctx, 0, job)

	// First attempt runs, then context is cancelled so backoff select exits
	if calls < 1 {
		t.Errorf("calls = %d, want >= 1", calls)
	}
}
