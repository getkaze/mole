package dashboard

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"

	"github.com/getkaze/mole/internal/store"
)

//go:embed templates/*.html templates/fragments/*.html
var templateFS embed.FS

//go:embed static/*
var staticFS embed.FS

// Config holds dashboard configuration.
type Config struct {
	GitHubClientID     string                `yaml:"github_client_id"`
	GitHubClientSecret string                `yaml:"github_client_secret"`
	SessionSecret      string                `yaml:"session_secret"`
	BaseURL            string                `yaml:"base_url"` // e.g. http://localhost:8080
	AllowedOrg         string                `yaml:"allowed_org"`
	Pricing            map[string][2]float64 // model -> [input, output] per 1M tokens
	Version            string
	Environment        string // "development" or "production"
}

// IsDev returns true when running in development mode.
func (c Config) IsDev() bool {
	return c.Environment == "development"
}

// Dashboard holds the handlers and dependencies.
type Dashboard struct {
	store  store.Store
	config Config
	pages  map[string]*template.Template
}

// New creates a new dashboard with parsed templates.
func New(s store.Store, cfg Config) (*Dashboard, error) {
	pages := make(map[string]*template.Template)

	// Template functions available to all templates
	funcMap := template.FuncMap{
		"formatTokens": formatTokens,
		"shortModule":  shortModule,
	}

	// Parse each page template with its own copy of the layout
	pageFiles := []string{"me.html", "team.html", "modules.html", "module.html", "developers.html", "developer.html", "costs.html", "about.html"}
	for _, page := range pageFiles {
		tmpl, err := template.New("").Funcs(funcMap).ParseFS(templateFS,
			"templates/layout.html",
			"templates/"+page,
		)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", page, err)
		}
		pages[page] = tmpl
	}

	// Login is standalone (no layout)
	login, err := template.ParseFS(templateFS, "templates/login.html")
	if err != nil {
		return nil, fmt.Errorf("parsing login: %w", err)
	}
	pages["login.html"] = login

	// Fragment templates
	fragments, err := template.New("").Funcs(funcMap).ParseFS(templateFS, "templates/fragments/*.html")
	if err != nil {
		return nil, fmt.Errorf("parsing fragments: %w", err)
	}
	pages["fragments"] = fragments

	return &Dashboard{
		store:  s,
		config: cfg,
		pages:  pages,
	}, nil
}

// RegisterRoutes adds dashboard routes to the given mux.
func (d *Dashboard) RegisterRoutes(mux *http.ServeMux) {
	// Static files
	staticSub, _ := fs.Sub(staticFS, "static")
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticSub))))

	// Auth
	mux.HandleFunc("GET /auth/login", d.handleAuthLogin)
	mux.HandleFunc("GET /auth/github", d.handleAuthGitHub)
	mux.HandleFunc("GET /auth/callback", d.handleAuthCallback)
	mux.HandleFunc("GET /auth/dev", d.handleAuthDev)
	mux.HandleFunc("GET /auth/logout", d.handleLogout)

	// Pages (auth required)
	mux.HandleFunc("GET /", d.requireAuth(d.handleRoot))
	mux.HandleFunc("GET /me", d.requireAuth(d.handleMe))
	mux.HandleFunc("GET /developers", d.requireAuth(d.handleDevelopers))
	mux.HandleFunc("GET /developers/{login}", d.requireAuth(d.handleDeveloper))
	mux.HandleFunc("GET /team", d.requireAuth(d.handleTeam))
	mux.HandleFunc("GET /modules", d.requireAuth(d.handleModules))
	mux.HandleFunc("GET /modules/{name...}", d.requireAuth(d.handleModule))

	// HTMX fragments — own dashboard
	mux.HandleFunc("GET /me/issues", d.requireAuth(d.handleMeIssues))
	mux.HandleFunc("GET /me/trends", d.requireAuth(d.handleMeTrends))
	mux.HandleFunc("GET /me/badges", d.requireAuth(d.handleMeBadges))

	// HTMX fragments — developer detail (reuses same logic, different target user)
	mux.HandleFunc("GET /developers/{login}/issues", d.requireAuth(d.handleDevIssues))
	mux.HandleFunc("GET /developers/{login}/trends", d.requireAuth(d.handleDevTrends))
	mux.HandleFunc("GET /developers/{login}/badges", d.requireAuth(d.handleDevBadges))

	// HTMX fragments — team
	mux.HandleFunc("GET /team/acceptance", d.requireAuth(d.handleTeamAcceptance))
	mux.HandleFunc("GET /team/distribution", d.requireAuth(d.handleTeamDistribution))
	mux.HandleFunc("GET /team/training", d.requireAuth(d.handleTeamTraining))

	// About
	mux.HandleFunc("GET /about", d.requireAuth(d.handleAbout))

	// Costs (admin only)
	mux.HandleFunc("GET /costs", d.requireAuth(d.handleCosts))
	mux.HandleFunc("GET /costs/breakdown", d.requireAuth(d.handleCostsBreakdown))
}
