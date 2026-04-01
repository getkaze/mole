# Changelog

All notable changes to mole will be documented here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
Versions follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Developer Experience

- **GitHub Gateway interface** — abstract all GitHub API calls behind a `Gateway` interface with `RemoteGateway` (production) and `LocalGateway` (fixtures from disk), enabling development and testing without a GitHub App (@mateusmetzker)
- **Local review mode** — `mole review --local <dir>` reads PR data from fixture files, calls Claude, and prints the formatted review to stdout + `output.md`, no GitHub needed (@mateusmetzker)
- **Development mode** — `server.environment: development` bypasses GitHub OAuth on the dashboard login, showing role-based test logins (Admin, Dev, Tech Lead, Manager) with a fixed `testuser` account (@mateusmetzker)
- **Test fixtures** — 12 sample PRs across 3 repos and 5 developers with intentional issues (SQL injection, XSS, credential leaks, etc.) for local review testing (@mateusmetzker)
- **Localized init** — `mole init --language pt-BR` generates architecture and conventions docs in the specified language; the language is also saved to `.mole/config.yaml` (@mateusmetzker)

### Dashboard

- **Score trend line chart** — replace bar chart with SVG line chart showing weekly score evolution with score labels, colored dots (green ≥90, yellow 70-89, red <70), delta tooltips, and period filters (30d/60d/90d). Aggregator now uses ISO week boundaries (Monday–Sunday) instead of rolling 7-day windows (@mateusmetzker)
- **Issues filter order** — reorder period filter buttons to 7d → 30d → 90d on both dashboard pages (@mateusmetzker)
- **Kaze design system** — replace amber/Recursive palette with neutral Inter-based design system matching the Kaze landing page: new color tokens, layered shadows, glassmorphism topbar, blue brand color for data viz, and refined component spacing (@mateusmetzker)
- **Remove Architect role** — consolidate access roles to Dev, Tech Lead, Manager, Admin (@mateusmetzker)
- **Module cards** — show aggregated metrics (summed issues/debt, averaged health) instead of duplicating per weekly period (@mateusmetzker)
- **Module detail** — weekly evolution chart and breakdown table when clicking a module card (@mateusmetzker)
- **Module card overflow** — long module names now truncate with ellipsis instead of breaking the card layout (@mateusmetzker)
- **Developer display names** — resolve GitHub profile names via cached `github_profiles` table, populated on OAuth login (@mateusmetzker)
- **Module links broken** — fix routing for module names containing slashes by using wildcard path matching (@mateusmetzker)
- **Module card names** — show only last 3 path segments instead of full module path for readability (@mateusmetzker)
- **Module file breakdown** — module detail page now shows issues and debt items grouped by file (@mateusmetzker)

### Configuration

- **Server-level defaults** — `language` and `personality` can now be set in `mole.yaml` under `defaults`, applying to all repos unless overridden by `.mole/config.yaml` (@mateusmetzker)

### PR Review

- **`/mole dig` command** — contextual review that clones the repo locally, creates a git worktree per PR, explores the codebase with Claude Haiku (multi-turn tool use: `get_file`, `search_code`, `list_dir`), then reviews with Opus using the collected context. Configurable via `repos.base_path` and `exploration.*` in `mole.yaml`. Falls back to diff-only if git is unavailable or clone fails. Clone status is posted as a PR comment in the configured language and personality (@mateusmetzker)
- **Remove suggestion severity** — drop the 🟢 suggestion level entirely; reviews now only report Critical (🔴) and Attention (🟡) issues, eliminating generic low-value noise and contradictory findings between reviews (@mateusmetzker)
- **Remove general suggestions** — remove the "Suggestions" section from PR review body; only line-specific issues remain (@mateusmetzker)
- **Rebalance score weights** — critical penalty reduced from 15 to 8 points; attention stays at 5 (@mateusmetzker)
- **Deep review on PR open** — first review when a PR is opened now uses deep review (Claude Opus) instead of standard (@mateusmetzker)
- **Localized LLM output** — review content (issues, summary) is now written in the configured language, not just the personality chrome (@mateusmetzker)
- **pt-BR accent fix** — severity labels and personality texts now use proper Portuguese diacritics (Crítico, Atenção) (@mateusmetzker)

## [0.1.0] — 2026-03-29

Initial public release (@mateusmetzker).

### PR Review

- **Standard reviews** — triggered automatically on PR open, or manually with `/mole review`
- **Deep reviews** — Claude Opus for thorough analysis with `/mole deep-review`
- **Ignore PRs** — skip reviews with `/mole ignore`
- **CLI reviews** — review any PR from terminal with `mole review owner/repo#123`
- **Bot personality** — 3 modes: `mole` (playful), `formal` (professional), `minimal` (terse)
- **Issue taxonomy** — Security, Bugs, Smells, Architecture, Performance, Style (with subcategories)
- **Quality score** — 0-100 per PR
- **Architecture validation** — layer enforcement rules via AST analysis
- **Security scanner** — AST-based detection of common vulnerabilities
- **Mermaid diagrams** — sequence and class diagrams in deep reviews
- **Reaction sync** — :+1: / :-1: on inline comments to confirm or mark false positives, synced hourly or via `mole sync`
- **False positive filtering** — confirmed false positives excluded from scores and metrics

### Dashboard

- Individual developer view — issue heat map, score trends, streaks, badges
- Team view — issue distribution, quality trends, training suggestions
- Module view — health score, tech debt tracking, grouped by repository
- Costs view — Claude API usage and estimated costs per model (admin only)
- About page — application info and version
- Gamification — streaks, badges, achievements
- Role-based access — Dev, Tech Lead, Manager, Admin
- i18n — Portuguese (default) and English, switchable via flag selector
- GitHub OAuth login with org-based access restriction
- HTMX-powered, no JavaScript framework
- Favicon with properly squared PNGs at 32, 96, and 180px

### CLI

- `mole serve` — start webhook server, worker pool, and metrics aggregator
- `mole migrate` — run database migrations (auto-runs on serve)
- `mole health` — check MySQL, Valkey, and GitHub connectivity
- `mole review owner/repo#123` — review a PR from the terminal
- `mole init /path/to/repo` — scan a repo and generate `.mole/` context files
- `mole sync` — sync reactions, recalculate scores, update metrics
- `mole admin set-role / list` — manage dashboard roles
- `mole update` — self-update from GitHub releases
- `mole version` — print version

### Configuration

- YAML config file (`mole.yaml`) with all fields overridable via `MOLE_` env vars
- Per-repo `.mole/` context files (architecture, conventions, decisions)
- Automatic context file generation via `mole init`

### Distribution

- Single static binary, no runtime dependencies
- Targets: Linux amd64, Linux arm64, macOS amd64, macOS arm64
- Docker image published to GHCR (`ghcr.io/getkaze/mole:main`)
- Docker Compose with MySQL and Valkey included
- Install script: `curl -fsSL https://getkaze.dev/mole/install.sh | sudo bash`

[Unreleased]: https://github.com/getkaze/mole/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/getkaze/mole/releases/tag/v0.1.0
