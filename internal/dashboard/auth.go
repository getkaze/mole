package dashboard

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	molestore "github.com/getkaze/mole/internal/store"
)

const (
	sessionCookieName = "mole_session"
	sessionMaxAge     = 7 * 24 * 3600 // 7 days
)

type sessionData struct {
	User      string    `json:"user"`
	ExpiresAt time.Time `json:"expires_at"`
}

func (d *Dashboard) handleAuthGitHub(w http.ResponseWriter, r *http.Request) {
	state := generateState()
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		MaxAge:   300,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	url := fmt.Sprintf(
		"https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s/auth/callback&state=%s&scope=read:user",
		d.config.GitHubClientID,
		d.config.BaseURL,
		state,
	)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (d *Dashboard) handleAuthCallback(w http.ResponseWriter, r *http.Request) {
	// Verify state
	stateCookie, err := r.Cookie("oauth_state")
	if err != nil || stateCookie.Value != r.URL.Query().Get("state") {
		http.Error(w, "invalid state", http.StatusBadRequest)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "missing code", http.StatusBadRequest)
		return
	}

	// Exchange code for token
	token, err := exchangeCode(d.config.GitHubClientID, d.config.GitHubClientSecret, code)
	if err != nil {
		slog.Error("oauth code exchange failed", "error", err)
		http.Error(w, "authentication failed", http.StatusInternalServerError)
		return
	}

	// Get user info
	username, err := getGitHubUser(token)
	if err != nil {
		slog.Error("failed to get github user", "error", err)
		http.Error(w, "authentication failed", http.StatusInternalServerError)
		return
	}

	// Ensure access record exists
	_, err = d.store.GetAccess(r.Context(), username)
	if err != nil {
		// Create default access
		d.store.UpsertAccess(r.Context(), &molestore.DashboardAccess{
			GitHubUser: username,
			Role:       "dev",
		})
	}

	// Set session cookie
	session := sessionData{
		User:      username,
		ExpiresAt: time.Now().Add(time.Duration(sessionMaxAge) * time.Second),
	}
	d.setSession(w, session)

	http.Redirect(w, r, "/me", http.StatusTemporaryRedirect)
}

func (d *Dashboard) handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func (d *Dashboard) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, err := d.getSession(r)
		if err != nil || session.User == "" || time.Now().After(session.ExpiresAt) {
			// For HTMX requests, return 401 instead of redirect
			if r.Header.Get("HX-Request") == "true" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			d.pages["login.html"].ExecuteTemplate(w, "login", nil)
			return
		}

		// Inject user into request context via a simple header trick
		r.Header.Set("X-Mole-User", session.User)
		next(w, r)
	}
}

func (d *Dashboard) getUser(r *http.Request) string {
	return r.Header.Get("X-Mole-User")
}

func (d *Dashboard) setSession(w http.ResponseWriter, session sessionData) {
	data, _ := json.Marshal(session)
	signed := d.sign(data)
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    signed,
		Path:     "/",
		MaxAge:   sessionMaxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func (d *Dashboard) getSession(r *http.Request) (*sessionData, error) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return nil, err
	}

	data, err := d.verify(cookie.Value)
	if err != nil {
		return nil, err
	}

	var session sessionData
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}

	return &session, nil
}

func (d *Dashboard) sign(data []byte) string {
	mac := hmac.New(sha256.New, []byte(d.config.SessionSecret))
	mac.Write(data)
	sig := hex.EncodeToString(mac.Sum(nil))
	encoded := base64.URLEncoding.EncodeToString(data)
	return encoded + "." + sig
}

func (d *Dashboard) verify(signed string) ([]byte, error) {
	parts := strings.SplitN(signed, ".", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid session format")
	}

	data, err := base64.URLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, err
	}

	mac := hmac.New(sha256.New, []byte(d.config.SessionSecret))
	mac.Write(data)
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(parts[1]), []byte(expected)) {
		return nil, fmt.Errorf("invalid signature")
	}

	return data, nil
}

func generateState() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func exchangeCode(clientID, clientSecret, code string) (string, error) {
	body := fmt.Sprintf("client_id=%s&client_secret=%s&code=%s", clientID, clientSecret, code)
	req, _ := http.NewRequest("POST", "https://github.com/login/oauth/access_token", strings.NewReader(body))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	respBody, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", err
	}
	if result.Error != "" {
		return "", fmt.Errorf("oauth error: %s", result.Error)
	}
	return result.AccessToken, nil
}

func getGitHubUser(token string) (string, error) {
	req, _ := http.NewRequest("GET", "https://api.github.com/user", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var user struct {
		Login string `json:"login"`
	}
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &user); err != nil {
		return "", err
	}
	if user.Login == "" {
		return "", fmt.Errorf("empty github login")
	}
	return user.Login, nil
}

