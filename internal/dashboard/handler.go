package dashboard

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	molestore "github.com/getkaze/mole/internal/store"
)

type pageData struct {
	User        string
	DisplayName string
	Page        string
	IsAdmin     bool
	RepoGroups []repoGroup
	Module     moduleView
	Developers []devOverview
	Developer  string
	Costs      *costsData
	Version    string
}

type costsData struct {
	Models       []modelCostView
	TotalCost    float64
	TotalInput   int64
	TotalOutput  int64
	TotalReviews int
	UniquePRs    int
	AvgReviewsPR string
	AvgCostPR    string
}

type modelCostView struct {
	Model       string
	Reviews     int
	InputTokens int64
	OutputTokens int64
	InputCost   string
	OutputCost  string
	TotalCost   string
}

type devOverview struct {
	Login       string
	Name        string
	Reviews     int
	AvgScore    float64
	Streak      int
	TopCategory string
}

type moduleFileView struct {
	FilePath    string
	TotalIssues int
	DebtItems   int
}

type moduleView struct {
	Repo        string
	ModuleName  string
	HealthScore float64
	TotalIssues int
	DebtItems   int
	Weeks       []moduleWeek
	Files       []moduleFileView
}

type moduleWeek struct {
	Label       string
	HealthScore float64
	TotalIssues int
	DebtItems   int
	Height      int
	Color       string
}

type repoGroup struct {
	Repo    string
	Modules []moduleView
}

func (d *Dashboard) isAdmin(r *http.Request) bool {
	access, _ := d.store.GetAccess(r.Context(), d.getUser(r))
	return access != nil && access.Role == "admin"
}

func (d *Dashboard) newPageData(r *http.Request, page string) pageData {
	return pageData{
		User:        d.getUser(r),
		DisplayName: d.getDisplayName(r),
		Page:        page,
		IsAdmin:     d.isAdmin(r),
	}
}

func (d *Dashboard) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/me", http.StatusTemporaryRedirect)
}

func (d *Dashboard) handleMe(w http.ResponseWriter, r *http.Request) {
	data := d.newPageData(r, "me")
	d.renderPage(w, "me.html", data)
}

func (d *Dashboard) handleTeam(w http.ResponseWriter, r *http.Request) {
	data := d.newPageData(r, "team")
	d.renderPage(w, "team.html", data)
}

func (d *Dashboard) handleModules(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	from := now.AddDate(0, 0, -30)

	metrics, err := d.store.ListAllModuleMetrics(r.Context(), "weekly", from, now)
	if err != nil {
		slog.Error("failed to get module metrics", "error", err)
	}

	// Group modules by repo, preserving order from the query (ORDER BY repo, module_name)
	groupMap := make(map[string]int) // repo -> index in groups slice
	var groups []repoGroup
	for _, m := range metrics {
		idx, ok := groupMap[m.Repo]
		if !ok {
			idx = len(groups)
			groupMap[m.Repo] = idx
			groups = append(groups, repoGroup{Repo: m.Repo})
		}
		groups[idx].Modules = append(groups[idx].Modules, moduleView{
			Repo:        m.Repo,
			ModuleName:  m.ModuleName,
			HealthScore: m.HealthScore,
			TotalIssues: m.TotalIssues,
			DebtItems:   m.DebtItems,
		})
	}

	data := d.newPageData(r, "modules")
	data.RepoGroups = groups
	d.renderPage(w, "modules.html", data)
}

func (d *Dashboard) handleModule(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	now := time.Now()
	from := now.AddDate(0, 0, -30)

	// name is "repo/module" encoded as path, but we receive it via {name} wildcard.
	// Since module_name can contain slashes, we need repo from query param.
	repo := r.URL.Query().Get("repo")

	metrics, err := d.store.GetModuleMetrics(r.Context(), repo, name, "weekly", from, now)
	if err != nil || len(metrics) == 0 {
		http.NotFound(w, r)
		return
	}

	// Aggregate totals across all weeks
	var totalIssues, totalDebt int
	var healthSum float64
	weeks := make([]moduleWeek, 0, len(metrics))
	for _, m := range metrics {
		totalIssues += m.TotalIssues
		totalDebt += m.DebtItems
		healthSum += m.HealthScore

		color := "green"
		if m.HealthScore < 60 {
			color = "red"
		} else if m.HealthScore < 80 {
			color = "yellow"
		}
		weeks = append(weeks, moduleWeek{
			Label:       m.PeriodStart.Format("02/01"),
			HealthScore: m.HealthScore,
			TotalIssues: m.TotalIssues,
			DebtItems:   m.DebtItems,
			Height:      int(m.HealthScore),
			Color:       color,
		})
	}
	avgHealth := healthSum / float64(len(metrics))

	// Group issues by file
	issues, err := d.store.GetIssuesByModule(r.Context(), repo, name, from, now)
	if err != nil {
		slog.Error("failed to get module issues", "error", err)
	}

	type fileAgg struct {
		Total int
		Debt  int
	}
	fileMap := make(map[string]*fileAgg)
	var fileOrder []string
	for _, issue := range issues {
		if issue.Validation == "false_positive" {
			continue
		}
		fp := issue.FilePath
		agg, ok := fileMap[fp]
		if !ok {
			agg = &fileAgg{}
			fileMap[fp] = agg
			fileOrder = append(fileOrder, fp)
		}
		agg.Total++
		if issue.Severity == "critical" {
			agg.Debt++
		}
	}
	files := make([]moduleFileView, 0, len(fileOrder))
	for _, fp := range fileOrder {
		agg := fileMap[fp]
		files = append(files, moduleFileView{
			FilePath:    fp,
			TotalIssues: agg.Total,
			DebtItems:   agg.Debt,
		})
	}

	data := d.newPageData(r, "modules")
	data.Module = moduleView{
		Repo:        repo,
		ModuleName:  name,
		HealthScore: avgHealth,
		TotalIssues: totalIssues,
		DebtItems:   totalDebt,
		Weeks:       weeks,
		Files:       files,
	}
	d.renderPage(w, "module.html", data)
}

// Developer Pages

func (d *Dashboard) handleDevelopers(w http.ResponseWriter, r *http.Request) {
	user := d.getUser(r)
	now := time.Now()
	from := now.AddDate(0, 0, -30)

	// Check role — only admin and tech_lead can see the list
	access, _ := d.store.GetAccess(r.Context(), user)
	role := "dev"
	if access != nil {
		role = access.Role
	}

	allMetrics, err := d.store.ListAllDevMetrics(r.Context(), "weekly", from, now)
	if err != nil {
		slog.Error("failed to list dev metrics", "error", err)
	}

	// Get latest per developer
	latestByDev := make(map[string]*molestore.DeveloperMetrics)
	for i := range allMetrics {
		m := &allMetrics[i]
		existing, ok := latestByDev[m.Developer]
		if !ok || m.PeriodStart.After(existing.PeriodStart) {
			latestByDev[m.Developer] = m
		}
	}

	// Resolve display names
	logins := make([]string, 0, len(latestByDev))
	for dev := range latestByDev {
		logins = append(logins, dev)
	}
	profiles, _ := d.store.GetGitHubProfiles(r.Context(), logins)

	var devs []devOverview
	for dev, m := range latestByDev {
		login := dev
		name := dev
		if displayName, ok := profiles[dev]; ok && displayName != "" {
			name = displayName
		}

		// Visibility rules
		if role == "dev" {
			// Devs only see themselves
			if dev != user {
				continue
			}
		} else if role == "manager" {
			// Manager sees no individual data
			continue
		} else if role != "admin" {
			// tech_lead: show only opted-in
			devAccess, _ := d.store.GetAccess(r.Context(), dev)
			if devAccess != nil && !devAccess.IndividualVisibility && dev != user {
				continue
			}
		}
		// admin sees everyone

		devs = append(devs, devOverview{
			Login:       login,
			Name:        name,
			Reviews:     m.TotalReviews,
			AvgScore:    m.AvgScore,
			Streak:      m.StreakCleanPRs,
			TopCategory: parseTopCategory(m.IssuesByCategory),
		})
	}

	data := d.newPageData(r, "developers")
	data.Developers = devs
	d.renderPage(w, "developers.html", data)
}

func (d *Dashboard) handleDeveloper(w http.ResponseWriter, r *http.Request) {
	user := d.getUser(r)
	target := r.PathValue("login")

	// Check access
	access, _ := d.store.GetAccess(r.Context(), user)
	role := "dev"
	if access != nil {
		role = access.Role
	}

	// Access control
	if target != user {
		switch role {
		case "admin":
			// ok
		case "tech_lead":
			targetAccess, _ := d.store.GetAccess(r.Context(), target)
			if targetAccess == nil || !targetAccess.IndividualVisibility {
				http.Error(w, "access denied", http.StatusForbidden)
				return
			}
		default:
			http.Error(w, "access denied", http.StatusForbidden)
			return
		}
	}

	data := d.newPageData(r, "developers")
	data.Developer = target
	d.renderPage(w, "developer.html", data)
}

// Developer HTMX fragments (same logic as /me but for any user)

func (d *Dashboard) handleDevIssues(w http.ResponseWriter, r *http.Request) {
	target := r.PathValue("login")
	d.renderIssuesFragment(w, r, target)
}

func (d *Dashboard) handleDevTrends(w http.ResponseWriter, r *http.Request) {
	target := r.PathValue("login")
	d.renderTrendsFragment(w, r, target)
}

func (d *Dashboard) handleDevBadges(w http.ResponseWriter, r *http.Request) {
	target := r.PathValue("login")
	d.renderBadgesFragment(w, r, target)
}

// HTMX Fragment Handlers

func (d *Dashboard) handleMeIssues(w http.ResponseWriter, r *http.Request) {
	d.renderIssuesFragment(w, r, d.getUser(r))
}

func (d *Dashboard) handleMeTrends(w http.ResponseWriter, r *http.Request) {
	d.renderTrendsFragment(w, r, d.getUser(r))
}

func (d *Dashboard) handleMeBadges(w http.ResponseWriter, r *http.Request) {
	d.renderBadgesFragment(w, r, d.getUser(r))
}

// Reusable fragment renderers (used by both /me and /developers/{login})

type categoryCount struct {
	Name    string
	Count   int
	Percent float64
}

type weekData struct {
	Label  string
	Score  float64
	Height float64
	Color  string
}

type badgeView struct {
	Icon string
	Name string
}

func (d *Dashboard) renderIssuesFragment(w http.ResponseWriter, r *http.Request, developer string) {
	now := time.Now()
	days := parsePeriod(r.URL.Query().Get("period"), 30)
	from := now.AddDate(0, 0, -days)

	issues, err := d.store.GetIssuesByDeveloper(r.Context(), developer, from, now)
	if err != nil {
		slog.Error("failed to get issues", "error", err)
	}

	counts := make(map[string]int)
	total := 0
	for _, i := range issues {
		if i.Validation == "false_positive" {
			continue
		}
		counts[i.Category]++
		total++
	}

	var categories []categoryCount
	for name, count := range counts {
		pct := 0.0
		if total > 0 {
			pct = float64(count) / float64(total) * 100
		}
		categories = append(categories, categoryCount{Name: name, Count: count, Percent: pct})
	}

	d.renderFragment(w, "heatmap.html", map[string]any{"Categories": categories})
}

func (d *Dashboard) renderTrendsFragment(w http.ResponseWriter, r *http.Request, developer string) {
	now := time.Now()
	from := now.AddDate(0, 0, -90)

	metrics, err := d.store.GetDevMetrics(r.Context(), developer, "weekly", from, now)
	if err != nil {
		slog.Error("failed to get dev metrics", "error", err)
	}

	var weeks []weekData
	for _, m := range metrics {
		color := "green"
		if m.AvgScore < 70 {
			color = "red"
		} else if m.AvgScore < 90 {
			color = "yellow"
		}
		weeks = append(weeks, weekData{
			Label:  m.PeriodStart.Format("Jan 2"),
			Score:  m.AvgScore,
			Height: m.AvgScore,
			Color:  color,
		})
	}

	d.renderFragment(w, "trends.html", map[string]any{"Weeks": weeks})
}

func (d *Dashboard) renderBadgesFragment(w http.ResponseWriter, r *http.Request, developer string) {
	streak, err := d.store.GetDevStreak(r.Context(), developer)
	if err != nil {
		slog.Error("failed to get streak", "error", err)
	}

	now := time.Now()
	from := now.AddDate(0, 0, -30)
	metrics, _ := d.store.GetDevMetrics(r.Context(), developer, "monthly", from, now)

	var badges []badgeView
	if len(metrics) > 0 && metrics[len(metrics)-1].Badges != "" {
		var badgeNames []string
		json.Unmarshal([]byte(metrics[len(metrics)-1].Badges), &badgeNames)
		for _, name := range badgeNames {
			badges = append(badges, badgeView{Icon: badgeIcon(name), Name: name})
		}
	}

	d.renderFragment(w, "badges.html", map[string]any{
		"Streak": streak,
		"Badges": badges,
	})
}

type devDistribution struct {
	Name        string
	Reviews     int
	AvgScore    float64
	TopCategory string
}

func (d *Dashboard) handleTeamDistribution(w http.ResponseWriter, r *http.Request) {
	user := d.getUser(r)
	now := time.Now()
	from := now.AddDate(0, 0, -30)

	allMetrics, err := d.store.ListAllDevMetrics(r.Context(), "weekly", from, now)
	if err != nil {
		slog.Error("failed to list dev metrics", "error", err)
	}

	// Get current user's role for visibility rules
	access, _ := d.store.GetAccess(r.Context(), user)
	role := "dev"
	if access != nil {
		role = access.Role
	}

	// Aggregate latest metrics per developer
	latestByDev := make(map[string]*molestore.DeveloperMetrics)
	for i := range allMetrics {
		m := &allMetrics[i]
		existing, ok := latestByDev[m.Developer]
		if !ok || m.PeriodStart.After(existing.PeriodStart) {
			latestByDev[m.Developer] = m
		}
	}

	var devs []devDistribution
	for dev, m := range latestByDev {
		name := dev
		// Admin sees everything — no anonymization
		if role == "admin" {
			// keep real name
		} else if role == "dev" || role == "manager" {
			// Anonymize for dev role and manager role
			if dev != user {
				name = "Developer " + dev[:1] + "***"
			}
		} else {
			// tech_lead: show name only if opted-in
			devAccess, _ := d.store.GetAccess(r.Context(), dev)
			if devAccess != nil && !devAccess.IndividualVisibility && dev != user {
				name = "Developer " + dev[:1] + "***"
			}
		}

		topCat := parseTopCategory(m.IssuesByCategory)
		devs = append(devs, devDistribution{
			Name:        name,
			Reviews:     m.TotalReviews,
			AvgScore:    m.AvgScore,
			TopCategory: topCat,
		})
	}

	d.renderFragment(w, "distribution.html", map[string]any{"Developers": devs})
}

func (d *Dashboard) handleTeamAcceptance(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	from := now.AddDate(0, 0, -30)

	rate, err := d.store.GetOverallAcceptanceRate(r.Context(), from, now)
	if err != nil {
		slog.Error("failed to get acceptance rate", "error", err)
	}

	d.renderFragment(w, "acceptance.html", map[string]any{"Rate": rate})
}

type trainingTopic struct {
	Category    string
	Subcategory string
	Count       int
}

func (d *Dashboard) handleTeamTraining(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	from := now.AddDate(0, 0, -30)

	patterns, err := d.store.ListTopIssuePatterns(r.Context(), from, now, 5)
	if err != nil {
		slog.Error("failed to list issue patterns", "error", err)
	}

	var topics []trainingTopic
	for _, p := range patterns {
		topics = append(topics, trainingTopic{
			Category:    p.Category,
			Subcategory: p.Subcategory,
			Count:       p.Count,
		})
	}

	d.renderFragment(w, "training.html", map[string]any{"Topics": topics})
}

// Costs page (admin only)

func (d *Dashboard) handleCosts(w http.ResponseWriter, r *http.Request) {
	if !d.isAdmin(r) {
		http.Error(w, "access denied", http.StatusForbidden)
		return
	}
	data := d.newPageData(r, "costs")
	d.renderPage(w, "costs.html", data)
}

func (d *Dashboard) handleCostsBreakdown(w http.ResponseWriter, r *http.Request) {
	if !d.isAdmin(r) {
		http.Error(w, "access denied", http.StatusForbidden)
		return
	}

	now := time.Now()
	days := parsePeriod(r.URL.Query().Get("period"), 30)
	from := now.AddDate(0, 0, -days)

	summaries, err := d.store.GetTokenUsageSummary(r.Context(), from, now, d.config.Pricing)
	if err != nil {
		slog.Error("failed to get token usage", "error", err)
	}

	costs := &costsData{}
	for _, s := range summaries {
		costs.Models = append(costs.Models, modelCostView{
			Model:        s.Model,
			Reviews:      s.Reviews,
			InputTokens:  s.InputTokens,
			OutputTokens: s.OutputTokens,
			InputCost:    fmt.Sprintf("%.2f", s.InputCost),
			OutputCost:   fmt.Sprintf("%.2f", s.OutputCost),
			TotalCost:    fmt.Sprintf("%.2f", s.TotalCost),
		})
		costs.TotalCost += s.TotalCost
		costs.TotalInput += s.InputTokens
		costs.TotalOutput += s.OutputTokens
		costs.TotalReviews += s.Reviews
	}

	uniquePRs, err := d.store.GetUniquePRCount(r.Context(), from, now)
	if err != nil {
		slog.Error("failed to get unique PR count", "error", err)
	}
	costs.UniquePRs = uniquePRs
	if uniquePRs > 0 {
		costs.AvgReviewsPR = fmt.Sprintf("%.1f", float64(costs.TotalReviews)/float64(uniquePRs))
		costs.AvgCostPR = fmt.Sprintf("%.2f", costs.TotalCost/float64(uniquePRs))
	} else {
		costs.AvgReviewsPR = "0"
		costs.AvgCostPR = "0.00"
	}

	d.renderFragment(w, "costs-breakdown.html", map[string]any{"Costs": costs})
}

func parseTopCategory(issuesByCategoryJSON string) string {
	if issuesByCategoryJSON == "" {
		return "-"
	}
	var counts map[string]int
	if err := json.Unmarshal([]byte(issuesByCategoryJSON), &counts); err != nil {
		return "-"
	}
	topCat := "-"
	topCount := 0
	for cat, count := range counts {
		if count > topCount {
			topCat = cat
			topCount = count
		}
	}
	return topCat
}

// About page

func (d *Dashboard) handleAbout(w http.ResponseWriter, r *http.Request) {
	data := d.newPageData(r, "about")
	data.Version = d.config.Version
	d.renderPage(w, "about.html", data)
}

// Helpers

func (d *Dashboard) renderPage(w http.ResponseWriter, page string, data any) {
	tmpl, ok := d.pages[page]
	if !ok {
		http.Error(w, "page not found", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		slog.Error("template render error", "page", page, "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

func (d *Dashboard) renderFragment(w http.ResponseWriter, name string, data any) {
	tmpl := d.pages["fragments"]
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, name, data); err != nil {
		slog.Error("fragment render error", "name", name, "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

func formatTokens(n int64) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	}
	return fmt.Sprintf("%d", n)
}

func shortModule(name string) string {
	parts := strings.Split(name, "/")
	if len(parts) <= 3 {
		return name
	}
	return strings.Join(parts[len(parts)-3:], "/")
}

func parsePeriod(s string, defaultDays int) int {
	switch s {
	case "7d":
		return 7
	case "30d":
		return 30
	case "90d":
		return 90
	default:
		return defaultDays
	}
}

func badgeIcon(name string) string {
	switch name {
	case "first_review":
		return "🎉"
	case "streak_5":
		return "🔥"
	case "streak_10":
		return "⚡"
	case "zero_critical_month":
		return "🛡️"
	case "quality_champion":
		return "🏆"
	default:
		return "🏅"
	}
}
