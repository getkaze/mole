package server

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/getkaze/mole/internal/store"
)

// --- Mocks ---

type fakeStore struct{ store.Store }

type mockPinger struct {
	err error
}

func (m *mockPinger) Ping(ctx context.Context) error { return m.err }

type mockStoreWithPing struct {
	store.Store
	pingErr   error
	ignored   map[string]bool
	ignoreErr error
}

func (m *mockStoreWithPing) Ping(ctx context.Context) error { return m.pingErr }
func (m *mockStoreWithPing) IsIgnored(ctx context.Context, repo string, pr int) (bool, error) {
	return false, nil
}
func (m *mockStoreWithPing) IgnorePR(ctx context.Context, repo string, pr int) error {
	if m.ignored != nil {
		m.ignored[repo] = true
	}
	return m.ignoreErr
}
func (m *mockStoreWithPing) SaveReview(ctx context.Context, r *store.Review) (int64, error) {
	return 1, nil
}
func (m *mockStoreWithPing) SaveIssues(ctx context.Context, reviewID int64, issues []store.Issue) ([]int64, error) {
	return nil, nil
}
func (m *mockStoreWithPing) UpdateIssueCommentID(ctx context.Context, issueID int64, commentID int64) error { return nil }
func (m *mockStoreWithPing) ValidateIssueByCommentID(ctx context.Context, githubCommentID int64, validation string, validatedBy string) error { return nil }
func (m *mockStoreWithPing) GetAcceptanceRate(ctx context.Context, dev string, from, to time.Time) (*store.AcceptanceRate, error) {
	return &store.AcceptanceRate{}, nil
}
func (m *mockStoreWithPing) GetOverallAcceptanceRate(ctx context.Context, from, to time.Time) (*store.AcceptanceRate, error) {
	return &store.AcceptanceRate{}, nil
}
func (m *mockStoreWithPing) GetIssuesByPR(ctx context.Context, repo string, pr int) ([]store.Issue, error) {
	return nil, nil
}
func (m *mockStoreWithPing) GetPendingValidationIssues(ctx context.Context, from, to time.Time) ([]store.Issue, error) {
	return nil, nil
}
func (m *mockStoreWithPing) GetReviewsWithPendingIssues(ctx context.Context, from, to time.Time) ([]store.Review, error) {
	return nil, nil
}
func (m *mockStoreWithPing) GetIssuesByDeveloper(ctx context.Context, dev string, from, to time.Time) ([]store.Issue, error) {
	return nil, nil
}
func (m *mockStoreWithPing) GetIssuesByModule(ctx context.Context, mod string, from, to time.Time) ([]store.Issue, error) {
	return nil, nil
}
func (m *mockStoreWithPing) UpsertInstallation(ctx context.Context, inst *store.Installation) error {
	return nil
}
func (m *mockStoreWithPing) AddRepository(ctx context.Context, repo *store.Repository) error {
	return nil
}
func (m *mockStoreWithPing) RemoveRepository(ctx context.Context, githubRepoID int64) error {
	return nil
}
func (m *mockStoreWithPing) GetInstallation(ctx context.Context, githubInstallID int64) (*store.Installation, error) {
	return &store.Installation{ID: 1}, nil
}
func (m *mockStoreWithPing) UpsertDevMetrics(ctx context.Context, met *store.DeveloperMetrics) error {
	return nil
}
func (m *mockStoreWithPing) GetDevMetrics(ctx context.Context, dev, pt string, from, to time.Time) ([]store.DeveloperMetrics, error) {
	return nil, nil
}
func (m *mockStoreWithPing) GetDevStreak(ctx context.Context, dev string) (int, error) {
	return 0, nil
}
func (m *mockStoreWithPing) ListAllDevMetrics(ctx context.Context, pt string, from, to time.Time) ([]store.DeveloperMetrics, error) {
	return nil, nil
}
func (m *mockStoreWithPing) UpsertModuleMetrics(ctx context.Context, met *store.ModuleMetrics) error {
	return nil
}
func (m *mockStoreWithPing) GetModuleMetrics(ctx context.Context, mod, pt string, from, to time.Time) ([]store.ModuleMetrics, error) {
	return nil, nil
}
func (m *mockStoreWithPing) ListAllModuleMetrics(ctx context.Context, pt string, from, to time.Time) ([]store.ModuleMetrics, error) {
	return nil, nil
}
func (m *mockStoreWithPing) GetAvgScoreByDeveloper(ctx context.Context, dev string, from, to time.Time) (float64, error) {
	return 0, nil
}
func (m *mockStoreWithPing) ListActiveDevelopers(ctx context.Context, from, to time.Time) ([]string, error) {
	return nil, nil
}
func (m *mockStoreWithPing) ListActiveModules(ctx context.Context, from, to time.Time) ([]string, error) {
	return nil, nil
}
func (m *mockStoreWithPing) ListTopIssuePatterns(ctx context.Context, from, to time.Time, limit int) ([]store.IssuePattern, error) {
	return nil, nil
}
func (m *mockStoreWithPing) GetAccess(ctx context.Context, user string) (*store.DashboardAccess, error) {
	return &store.DashboardAccess{GitHubUser: user, Role: "dev"}, nil
}
func (m *mockStoreWithPing) UpsertAccess(ctx context.Context, access *store.DashboardAccess) error {
	return nil
}
func (m *mockStoreWithPing) Close() error { return nil }

// --- Health Endpoint Tests ---

func TestHealthCheck_AllHealthy(t *testing.T) {
	h := &HealthChecker{
		Store: &mockStoreWithPing{},
		Queue: &mockPinger{},
	}

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	h.Handle(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body["mysql"] != "ok" {
		t.Errorf("mysql = %q, want %q", body["mysql"], "ok")
	}
	if body["valkey"] != "ok" {
		t.Errorf("valkey = %q, want %q", body["valkey"], "ok")
	}
}

func TestHealthCheck_MySQLDown(t *testing.T) {
	h := &HealthChecker{
		Store: &mockStoreWithPing{pingErr: errors.New("connection refused")},
		Queue: &mockPinger{},
	}

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	h.Handle(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}

	var body map[string]string
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["mysql"] != "error" {
		t.Errorf("mysql = %q, want %q", body["mysql"], "error")
	}
	if body["valkey"] != "ok" {
		t.Errorf("valkey = %q, want %q", body["valkey"], "ok")
	}
}

func TestHealthCheck_ValkeyDown(t *testing.T) {
	h := &HealthChecker{
		Store: &mockStoreWithPing{},
		Queue: &mockPinger{err: errors.New("connection refused")},
	}

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	h.Handle(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}

	var body map[string]string
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["valkey"] != "error" {
		t.Errorf("valkey = %q, want %q", body["valkey"], "error")
	}
}

func TestHealthCheck_BothDown(t *testing.T) {
	h := &HealthChecker{
		Store: &mockStoreWithPing{pingErr: errors.New("down")},
		Queue: &mockPinger{err: errors.New("down")},
	}

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	h.Handle(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

func TestHealthCheck_ContentType(t *testing.T) {
	h := &HealthChecker{
		Store: &mockStoreWithPing{},
		Queue: &mockPinger{},
	}

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	h.Handle(w, req)

	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}
}

// --- Webhook Tests ---

func TestWebhookHandler_InvalidMethod(t *testing.T) {
	h := &WebhookHandler{webhookSecret: []byte("secret")}

	req := httptest.NewRequest(http.MethodGet, "/webhook", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestWebhookHandler_InvalidSignature(t *testing.T) {
	h := &WebhookHandler{webhookSecret: []byte("secret")}

	body := []byte(`{"action":"opened"}`)
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set("X-Hub-Signature-256", "sha256=invalid")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestWebhookHandler_ValidSignature_PingEvent(t *testing.T) {
	secret := "test-secret"
	h := &WebhookHandler{webhookSecret: []byte(secret), store: &fakeStore{}}

	body := []byte(`{"action":"opened"}`)
	sig := computeHMAC(body, secret)

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set("X-Hub-Signature-256", "sha256="+sig)
	req.Header.Set("X-GitHub-Event", "ping")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestWebhookHandler_MaxBodySize(t *testing.T) {
	h := &WebhookHandler{webhookSecret: []byte("secret")}

	bigBody := strings.NewReader(strings.Repeat("x", 11<<20))
	req := httptest.NewRequest(http.MethodPost, "/webhook", bigBody)
	req.Header.Set("X-Hub-Signature-256", "sha256=anything")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d for oversized body", w.Code, http.StatusBadRequest)
	}
}

func TestWebhookHandler_EmptyBody(t *testing.T) {
	secret := "s"
	h := &WebhookHandler{webhookSecret: []byte(secret)}

	body := []byte("")
	sig := computeHMAC(body, secret)

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set("X-Hub-Signature-256", "sha256="+sig)
	req.Header.Set("X-GitHub-Event", "unknown_event")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)
	// Valid signature on empty body, unknown event → 200 OK
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func computeHMAC(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}
