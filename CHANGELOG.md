# Changelog

All notable changes to mole will be documented here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
Versions follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Dashboard

- **Module cards** — show aggregated metrics (summed issues/debt, averaged health) instead of duplicating per weekly period
- **Module detail** — weekly evolution chart and breakdown table when clicking a module card
- **Module card overflow** — long module names now truncate with ellipsis instead of breaking the card layout
- **Developer display names** — resolve GitHub profile names via cached `github_profiles` table, populated on OAuth login

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
