# Architecture

Kite is a self-hosted AI code reviewer that runs as a GitHub App. Single Go binary, no external AI infra needed — just an Anthropic API key.

## High-Level Flow

```
GitHub webhook → HTTP server → Valkey queue → Worker pool → Claude API → GitHub review
```

## Package Structure

```
cmd/kite/          — CLI entry point (Cobra): serve, migrate, health, review, version
internal/
  config/          — YAML config + env overrides
  server/          — HTTP server, webhook handler, signature validation
  queue/           — Valkey-backed job queue with dedup (72h TTL)
  worker/          — Worker pool with retry + exponential backoff (3 attempts)
  review/          — Review orchestration: split diffs, call LLM, format, validate
  llm/             — LLM provider interface + Claude implementation
  github/          — GitHub API: diff fetch, review posting, reactions, context loading
  store/           — MySQL: review history, ignored PRs
  migrate/         — Embedded SQL migrations (golang-migrate)
  metrics/         — Prometheus counters and histograms
  i18n/            — Localization (en, pt-BR)
```

## Key Design Decisions

- **Single binary** — `kite serve` runs HTTP server + worker pool in one process
- **Queue-first** — Webhooks enqueue jobs immediately, workers process async
- **Token budget splitting** — Large PRs are split into groups that fit Claude's context window (150K tokens)
- **Validator** — LLM line numbers are validated against actual diff hunk ranges before posting
- **Per-repo context** — Repos can add `.kite/*.md` files for project-specific review context
