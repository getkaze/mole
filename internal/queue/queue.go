package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/rueidis"
)

const (
	jobsKey       = "mole:queue:jobs"
	deadLetterKey = "mole:queue:deadletter"
	dedupPrefix   = "mole:dedup:"
	dedupTTL      = 72 * time.Hour
)

type Job struct {
	ID         string    `json:"id"`
	Type       string    `json:"type"` // "standard" or "deep"
	Repo       string    `json:"repo"` // "owner/repo"
	PRNumber   int       `json:"pr_number"`
	InstallID  int64     `json:"install_id"`
	DeliveryID string    `json:"delivery_id"`
	CommentID  int64     `json:"comment_id,omitempty"`
	Attempts   int       `json:"attempts"`
	CreatedAt  time.Time `json:"created_at"`
}

type Queue struct {
	client rueidis.Client
}

func New(addr string) (*Queue, error) {
	client, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress: []string{addr},
	})
	if err != nil {
		return nil, fmt.Errorf("connecting to valkey: %w", err)
	}
	return &Queue{client: client}, nil
}

func (q *Queue) Ping(ctx context.Context) error {
	cmd := q.client.B().Ping().Build()
	return q.client.Do(ctx, cmd).Error()
}

func (q *Queue) Close() {
	q.client.Close()
}

func (q *Queue) Enqueue(ctx context.Context, job Job) error {
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("marshaling job: %w", err)
	}
	cmd := q.client.B().Lpush().Key(jobsKey).Element(string(data)).Build()
	return q.client.Do(ctx, cmd).Error()
}

func (q *Queue) Dequeue(ctx context.Context, timeout time.Duration) (*Job, error) {
	cmd := q.client.B().Brpop().Key(jobsKey).Timeout(timeout.Seconds()).Build()
	result, err := q.client.Do(ctx, cmd).AsStrSlice()
	if err != nil {
		if rueidis.IsRedisNil(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("dequeuing job: %w", err)
	}
	if len(result) < 2 {
		return nil, nil
	}

	var job Job
	if err := json.Unmarshal([]byte(result[1]), &job); err != nil {
		return nil, fmt.Errorf("unmarshaling job: %w", err)
	}
	return &job, nil
}

func (q *Queue) DeadLetter(ctx context.Context, job Job, jobErr error) error {
	job.Attempts++
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("marshaling dead letter: %w", err)
	}
	cmd := q.client.B().Lpush().Key(deadLetterKey).Element(string(data)).Build()
	return q.client.Do(ctx, cmd).Error()
}

func (q *Queue) Len(ctx context.Context) (int64, error) {
	cmd := q.client.B().Llen().Key(jobsKey).Build()
	return q.client.Do(ctx, cmd).AsInt64()
}

func (q *Queue) IsDuplicate(ctx context.Context, deliveryID string) (bool, error) {
	cmd := q.client.B().Exists().Key(dedupPrefix + deliveryID).Build()
	n, err := q.client.Do(ctx, cmd).AsInt64()
	if err != nil {
		return false, fmt.Errorf("checking dedup: %w", err)
	}
	return n > 0, nil
}

func (q *Queue) MarkProcessed(ctx context.Context, deliveryID string) error {
	cmd := q.client.B().Set().Key(dedupPrefix+deliveryID).Value("1").Ex(dedupTTL).Build()
	return q.client.Do(ctx, cmd).Error()
}
