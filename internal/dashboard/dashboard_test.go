package dashboard

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/getkaze/mole/internal/store"
)

// --- Mock Store ---

type mockStore struct {
	store.Store
}

func (m *mockStore) Ping(ctx context.Context) error                                 { return nil }
func (m *mockStore) Close() error                                                   { return nil }
func (m *mockStore) SaveReview(ctx context.Context, r *store.Review) (int64, error) { return 1, nil }
func (m *mockStore) IsIgnored(ctx context.Context, repo string, pr int) (bool, error) {
	return false, nil
}
func (m *mockStore) IgnorePR(ctx context.Context, repo string, pr int) error { return nil }
func (m *mockStore) SaveIssues(ctx context.Context, reviewID int64, issues []store.Issue) ([]int64, error) {
	return nil, nil
}
func (m *mockStore) UpdateIssueCommentID(ctx context.Context, issueID int64, commentID int64) error {
	return nil
}
func (m *mockStore) ValidateIssueByCommentID(ctx context.Context, githubCommentID int64, validation string, validatedBy string) error {
	return nil
}
func (m *mockStore) GetAcceptanceRate(ctx context.Context, dev string, from, to time.Time) (*store.AcceptanceRate, error) {
	return &store.AcceptanceRate{}, nil
}
func (m *mockStore) GetOverallAcceptanceRate(ctx context.Context, from, to time.Time) (*store.AcceptanceRate, error) {
	return &store.AcceptanceRate{}, nil
}
func (m *mockStore) GetPendingValidationIssues(ctx context.Context, from, to time.Time) ([]store.Issue, error) {
	return nil, nil
}
func (m *mockStore) GetIssuesByPR(ctx context.Context, repo string, pr int) ([]store.Issue, error) {
	return nil, nil
}
func (m *mockStore) GetReviewsWithPendingIssues(ctx context.Context, from, to time.Time) ([]store.Review, error) {
	return nil, nil
}
func (m *mockStore) GetIssuesByDeveloper(ctx context.Context, dev string, from, to time.Time) ([]store.Issue, error) {
	return nil, nil
}
func (m *mockStore) GetIssuesByModule(ctx context.Context, repo, mod string, from, to time.Time) ([]store.Issue, error) {
	return nil, nil
}
func (m *mockStore) UpsertInstallation(ctx context.Context, inst *store.Installation) error {
	return nil
}
func (m *mockStore) AddRepository(ctx context.Context, repo *store.Repository) error { return nil }
func (m *mockStore) RemoveRepository(ctx context.Context, githubRepoID int64) error  { return nil }
func (m *mockStore) GetInstallation(ctx context.Context, id int64) (*store.Installation, error) {
	return &store.Installation{ID: 1}, nil
}
func (m *mockStore) UpsertDevMetrics(ctx context.Context, met *store.DeveloperMetrics) error {
	return nil
}
func (m *mockStore) GetDevMetrics(ctx context.Context, dev, pt string, from, to time.Time) ([]store.DeveloperMetrics, error) {
	return nil, nil
}
func (m *mockStore) GetDevStreak(ctx context.Context, dev string) (int, error) { return 0, nil }
func (m *mockStore) UpsertModuleMetrics(ctx context.Context, met *store.ModuleMetrics) error {
	return nil
}
func (m *mockStore) GetModuleMetrics(ctx context.Context, repo, mod, pt string, from, to time.Time) ([]store.ModuleMetrics, error) {
	return nil, nil
}
func (m *mockStore) GetAccess(ctx context.Context, user string) (*store.DashboardAccess, error) {
	return &store.DashboardAccess{GitHubUser: user, Role: "dev"}, nil
}
func (m *mockStore) UpsertAccess(ctx context.Context, access *store.DashboardAccess) error {
	return nil
}
func (m *mockStore) ListAllDevMetrics(ctx context.Context, pt string, from, to time.Time) ([]store.DeveloperMetrics, error) {
	return nil, nil
}
func (m *mockStore) ListAllModuleMetrics(ctx context.Context, pt string, from, to time.Time) ([]store.ModuleMetrics, error) {
	return nil, nil
}
func (m *mockStore) GetAvgScoreByDeveloper(ctx context.Context, dev string, from, to time.Time) (float64, error) {
	return 0, nil
}
func (m *mockStore) ListActiveDevelopers(ctx context.Context, from, to time.Time) ([]string, error) {
	return nil, nil
}
func (m *mockStore) ListActiveModules(ctx context.Context, from, to time.Time) ([]store.RepoModule, error) {
	return nil, nil
}
func (m *mockStore) ListTopIssuePatterns(ctx context.Context, from, to time.Time, limit int) ([]store.IssuePattern, error) {
	return nil, nil
}

// --- Tests ---

func newTestDashboard(t *testing.T) *Dashboard {
	t.Helper()
	d, err := New(&mockStore{}, Config{
		GitHubClientID:     "test-id",
		GitHubClientSecret: "test-secret",
		SessionSecret:      "test-session-secret-32chars!!!!",
		BaseURL:            "http://localhost:8080",
	})
	if err != nil {
		t.Fatalf("failed to create dashboard: %v", err)
	}
	return d
}

func TestNew_ParsesTemplates(t *testing.T) {
	d := newTestDashboard(t)
	if d.pages == nil {
		t.Fatal("pages should be parsed")
	}
	if _, ok := d.pages["me.html"]; !ok {
		t.Error("should have me.html page")
	}
	if _, ok := d.pages["login.html"]; !ok {
		t.Error("should have login.html page")
	}
	if _, ok := d.pages["documentation.html"]; !ok {
		t.Error("should have documentation.html page")
	}
	if _, ok := d.pages["fragments"]; !ok {
		t.Error("should have fragments template")
	}
}

func TestLoginPage_UnauthenticatedAccess(t *testing.T) {
	d := newTestDashboard(t)

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	w := httptest.NewRecorder()

	handler := d.requireAuth(d.handleMe)
	handler(w, req)

	// Should redirect to login page
	if w.Code != http.StatusTemporaryRedirect {
		t.Errorf("status = %d, want 307 (redirect to login)", w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "/auth/login" {
		t.Errorf("redirect location = %q, want /auth/login", loc)
	}
}

func TestHTMXFragment_UnauthenticatedReturns401(t *testing.T) {
	d := newTestDashboard(t)

	req := httptest.NewRequest(http.MethodGet, "/me/issues", nil)
	req.Header.Set("HX-Request", "true")
	w := httptest.NewRecorder()

	handler := d.requireAuth(d.handleMeIssues)
	handler(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401 for HTMX without auth", w.Code)
	}
}

func TestSessionSignVerify(t *testing.T) {
	d := newTestDashboard(t)
	data := []byte(`{"user":"alice","expires_at":"2030-01-01T00:00:00Z"}`)

	signed := d.sign(data)
	verified, err := d.verify(signed)
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if string(verified) != string(data) {
		t.Errorf("data mismatch: got %q, want %q", verified, data)
	}
}

func TestSessionVerify_InvalidSignature(t *testing.T) {
	d := newTestDashboard(t)
	_, err := d.verify("baddata.badsig")
	if err == nil {
		t.Error("should fail for invalid signature")
	}
}

func TestSessionVerify_InvalidFormat(t *testing.T) {
	d := newTestDashboard(t)
	_, err := d.verify("noseparator")
	if err == nil {
		t.Error("should fail for missing separator")
	}
}

func TestAuthenticatedAccess(t *testing.T) {
	d := newTestDashboard(t)

	// Create a valid session
	session := sessionData{
		User:      "testuser",
		ExpiresAt: time.Now().Add(time.Hour),
	}
	w := httptest.NewRecorder()
	d.setSession(w, session)

	// Extract the cookie
	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("no session cookie set")
	}

	// Make authenticated request
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.AddCookie(cookies[0])
	w2 := httptest.NewRecorder()

	handler := d.requireAuth(d.handleMe)
	handler(w2, req)

	if w2.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 for authenticated user", w2.Code)
	}
	body := w2.Body.String()
	if len(body) == 0 {
		t.Error("should render dashboard content")
	}
}

func TestDocumentationPage_AuthenticatedAccess(t *testing.T) {
	d := newTestDashboard(t)

	session := sessionData{
		User:      "testuser",
		ExpiresAt: time.Now().Add(time.Hour),
	}
	w := httptest.NewRecorder()
	d.setSession(w, session)

	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("no session cookie set")
	}

	mux := http.NewServeMux()
	d.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/documentation", nil)
	req.AddCookie(cookies[0])
	w2 := httptest.NewRecorder()
	mux.ServeHTTP(w2, req)

	if w2.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 for authenticated user", w2.Code)
	}
	body := w2.Body.String()
	if !strings.Contains(body, "/mole review") {
		t.Error("should render documentation command")
	}
	if !strings.Contains(body, "sidebar-item active") {
		t.Error("should render documentation sidebar item as active")
	}
}

func TestDocumentationPage_UnauthenticatedAccess(t *testing.T) {
	d := newTestDashboard(t)
	mux := http.NewServeMux()
	d.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/documentation", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusTemporaryRedirect {
		t.Errorf("status = %d, want 307 (redirect to login)", w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "/auth/login" {
		t.Errorf("redirect location = %q, want /auth/login", loc)
	}
}

func TestStaticFiles(t *testing.T) {
	d := newTestDashboard(t)
	mux := http.NewServeMux()
	d.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/static/style.css", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 for static CSS", w.Code)
	}
}

func TestParsePeriod(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"7d", 7},
		{"30d", 30},
		{"90d", 90},
		{"", 30},
		{"invalid", 30},
	}
	for _, tt := range tests {
		if got := parsePeriod(tt.input, 30); got != tt.want {
			t.Errorf("parsePeriod(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}
