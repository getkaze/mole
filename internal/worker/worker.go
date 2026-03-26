package worker

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/getkaze/kite/internal/metrics"
	"github.com/getkaze/kite/internal/queue"
)

type ReviewFunc func(ctx context.Context, job queue.Job) error

type Pool struct {
	queue     *queue.Queue
	reviewFn  ReviewFunc
	count     int
	wg        sync.WaitGroup
	cancel    context.CancelFunc
}

func NewPool(q *queue.Queue, reviewFn ReviewFunc, count int) *Pool {
	return &Pool{
		queue:    q,
		reviewFn: reviewFn,
		count:    count,
	}
}

func (p *Pool) Start(ctx context.Context) {
	ctx, p.cancel = context.WithCancel(ctx)

	for i := range p.count {
		p.wg.Add(1)
		go p.run(ctx, i)
	}

	slog.Info("worker pool started", "count", p.count)
}

func (p *Pool) Stop() {
	slog.Info("stopping worker pool")
	p.cancel()
	p.wg.Wait()
	slog.Info("worker pool stopped")
}

func (p *Pool) run(ctx context.Context, id int) {
	defer p.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		job, err := p.queue.Dequeue(ctx, 5*time.Second)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Error("dequeue error", "worker", id, "error", err)
			time.Sleep(time.Second)
			continue
		}
		if job == nil {
			continue
		}

		if depth, err := p.queue.Len(ctx); err == nil {
			metrics.QueueDepth.Set(float64(depth))
		}

		slog.Info("processing job", "worker", id, "job_id", job.ID, "repo", job.Repo, "pr", job.PRNumber, "type", job.Type)
		p.processWithRetry(ctx, id, job)
	}
}

func (p *Pool) processWithRetry(ctx context.Context, workerID int, job *queue.Job) {
	maxAttempts := 3
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		job.Attempts = attempt
		err := p.reviewFn(ctx, *job)
		if err == nil {
			slog.Info("job completed", "worker", workerID, "job_id", job.ID)
			return
		}

		slog.Warn("job failed", "worker", workerID, "job_id", job.ID, "attempt", attempt, "error", err)

		if attempt < maxAttempts {
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}
		}
	}

	slog.Error("job exhausted retries, moving to dead letter", "job_id", job.ID)
	if err := p.queue.DeadLetter(ctx, *job, nil); err != nil {
		slog.Error("failed to dead letter job", "job_id", job.ID, "error", err)
	}
}
