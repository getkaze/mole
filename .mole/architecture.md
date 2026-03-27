# architecture.md

## What It Does
Mole is an AI-powered GitHub pull request reviewer that acts as a GitHub App: it receives webhook events, queues review jobs, calls Claude (Anthropic) LLM to analyze diffs, and posts inline comments back to the PR.

## High-Level Flow

```
GitHub Webhook → server (HTTP) → queue (Valkey/Redis)
                                        ↓
                               worker pool (N goroutines)
                                        ↓
                         review.Service.Execute()
                           ├── github.FetchDiff()
                           ├── github.LoadContext()   (.mole/ docs)
                           ├── github.LoadRepoConfig() (.mole/config.yaml)
                           ├── arch.Validate()        (static layer checks)
                           ├── llm.Claude.Review()    (Claude API streaming)
                           └── github.PostReview()    (inline comments)
                                        ↓
                               store (MySQL) — issues, metrics, access
                                        ↓
                            aggregator (hourly) → developer/module metrics
                                        ↓
                            dashboard (HTMX, GitHub OAuth) — web UI
```

## Package/Module Structure

| Package | Purpose |
|---|---|
| `cmd/mole` | CLI entry point: `serve`, `migrate`, `review`, `init`, `health` subcommands via Cobra |
| `internal/server` | HTTP server: webhook ingestion, route registration interface (`RouteRegistrar`) |
| `internal/queue` | Valkey-backed job queue; `Job` struct with type, repo, PR number, install ID |
| `internal/worker` | Worker pool; calls `svc.Execute` per job |
| `internal/review` | Core review orchestration: fetch diff → load context → LLM → post comments → store |
| `internal/llm` | Claude client, prompt building, diff line numbering, JSON response parsing |
| `internal/github` | GitHub App client factory, diff fetching, context loading, repo config, posting reviews |
| `internal/store` | MySQL-backed persistence for reviews, issues, metrics, dashboard access |
| `internal/aggregator` | Periodic (hourly) metric aggregation: dev metrics, module health, badges |
| `internal/dashboard` | HTMX web UI with GitHub OAuth; pages for "me", "team", "modules" |
| `internal/config` | YAML config loading with env overrides and validation |
| `internal/migrate` | Embedded SQL migrations via `migrations/` directory |
| `internal/scan` | Repo scanner for `init` command: detects language, framework, generates context docs |
| `internal/arch` | Static Go import analysis for architecture layer violation detection |
| `internal/ast` | Go AST parsing to generate Mermaid class diagrams |
| `internal/i18n` | Static translation strings (en, pt-BR) |
| `internal/score` | PR scoring logic |
| `internal/security` | Security-focused analysis helpers |
| `internal/personality` | Personality/tone configuration for review output |
| `internal/metrics` | Internal metric collection |

## Key Design Decisions
- **Dependency injection**: services receive interfaces (e.g., `store.Store`) via constructors (`New*`)
- **GitHub App**: per-installation JWT clients cached in `ClientFactory` with mutex
- **Streaming LLM**: Claude API called with streaming, accumulated into final message
- **HTMX dashboard**: server-side rendered HTML with fragment endpoints; no separate SPA
- **Context files**: repos place `.mole/architecture.md`, `.mole/conventions.md`, `.mole/config.yaml` — loaded per review
- **`RouteRegistrar` interface**: optional dashboard plugged into the HTTP server

## External Dependencies
- `github.com/anthropics/anthropic-sdk-go` — LLM
- `github.com/google/go-github/v72` — GitHub API
- `github.com/bradleyfalzon/ghinstallation/v2` — GitHub App auth
- `github.com/spf13/cobra` — CLI
- `gopkg.in/yaml.v3` — config parsing
- MySQL (via DSN) + Valkey/Redis (queue)
