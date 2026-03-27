package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	ReviewsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "mole_reviews_total",
		Help: "Total number of reviews performed.",
	}, []string{"type", "status"})

	ReviewDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "mole_review_duration_seconds",
		Help:    "Duration of review processing in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"type"})

	TokensUsed = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "mole_tokens_used_total",
		Help: "Total tokens consumed by LLM calls.",
	}, []string{"model", "direction"})

	WebhooksReceived = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "mole_webhooks_received_total",
		Help: "Total webhook events received.",
	}, []string{"event"})

	QueueDepth = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "mole_queue_depth",
		Help: "Current number of jobs in the review queue.",
	})
)
