package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/getkaze/kite/internal/queue"
	"github.com/getkaze/kite/internal/store"
)

type Server struct {
	httpServer *http.Server
	store      store.Store
	queue      *queue.Queue
}

type Pinger interface {
	Ping(ctx context.Context) error
}

type HealthChecker struct {
	Store store.Store
	Queue Pinger
}

func New(port int, webhookSecret string, q *queue.Queue, s store.Store) *Server {
	mux := http.NewServeMux()

	webhook := NewWebhookHandler(webhookSecret, q, s)
	mux.Handle("POST /webhook", webhook)

	health := &HealthChecker{Store: s, Queue: q}
	mux.HandleFunc("GET /health", health.Handle)
	mux.Handle("GET /metrics", promhttp.Handler())

	return &Server{
		httpServer: &http.Server{
			Addr:         fmt.Sprintf(":%d", port),
			Handler:      mux,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
		},
		store: s,
		queue: q,
	}
}

func (s *Server) Start() error {
	slog.Info("starting http server", "addr", s.httpServer.Addr)
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("http server: %w", err)
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	slog.Info("shutting down http server")
	return s.httpServer.Shutdown(ctx)
}

func (h *HealthChecker) Handle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	status := http.StatusOK

	mysqlStatus := "ok"
	if err := h.Store.Ping(ctx); err != nil {
		mysqlStatus = "error"
		status = http.StatusServiceUnavailable
	}

	valkeyStatus := "ok"
	if err := h.Queue.Ping(ctx); err != nil {
		valkeyStatus = "error"
		status = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	fmt.Fprintf(w, `{"mysql":%q,"valkey":%q}`, mysqlStatus, valkeyStatus)
}
