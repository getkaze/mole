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

	"github.com/getkaze/kite/internal/store"
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
func (m *mockStoreWithPing) SaveReview(ctx context.Context, r *store.Review) error { return nil }
func (m *mockStoreWithPing) Close() error                                          { return nil }

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
