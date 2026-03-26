<div align="center">

  <img src="logo.svg" alt="kite" width="48" height="48"/>

  # kite

  **AI-powered PR reviews. Self-hosted. One binary.**

  <br/>

  [![Go](https://img.shields.io/badge/Go-1.26-00ADD8?style=flat-square&logo=go&logoColor=white)](https://golang.org)
  [![License](https://img.shields.io/badge/license-MIT-green?style=flat-square)](LICENSE)
  [![Platform](https://img.shields.io/badge/platform-linux%20%7C%20macOS-lightgrey?style=flat-square)](https://github.com/getkaze/kite)

  <br/>

  [What is Kite](#what-is-kite) · [Prerequisites](#prerequisites) · [Install](#install) · [Setup](#setup) · [Usage](#usage) · [How It Works](#how-it-works) · [Context Files](#context-files) · [Config Reference](#config-reference) · [Stack](#stack) · [Build](#build)

</div>

---

## What is Kite

**Kite** (the bird that soars above, spotting what others miss) is an open-source, self-hosted AI code reviewer. Install it as a GitHub App, point it at your repos, and every PR gets an automated review powered by Claude.

- **Standard reviews** — triggered automatically on PR open, or manually with `/kite review`
- **Deep reviews** — use Claude Opus for thorough analysis with `/kite deep-review`
- **Ignore PRs** — skip reviews with `/kite ignore`
- **CLI reviews** — review any PR from your terminal with `kite review owner/repo#123`

---

## Prerequisites

- **GitHub App** — you create one in your GitHub account (Kite runs as a GitHub App)
- **MySQL 8.0+** — stores review history and ignored PRs
- **Valkey 7.0+** (or Redis) — job queue and webhook dedup
- **Anthropic API key** — for Claude-powered reviews

---

## Install

```bash
curl -fsSL https://getkaze.dev/kite/install.sh | sudo bash
```

Or download the binary from [Releases](https://github.com/getkaze/kite/releases) and place it in your `PATH`.

---

## Setup

### 1. Create a GitHub App

Go to [github.com/settings/apps/new](https://github.com/settings/apps/new) and create a new app:

| Setting | Value |
|---------|-------|
| Webhook URL | `https://your-server.com/webhook` |
| Webhook secret | Generate a strong secret |
| Permissions | Pull requests (read & write), Contents (read) |
| Events | Pull request, Issue comment |

Download the private key and note the App ID.

### 2. Configure

```bash
cp kite.yaml.example kite.yaml
```

Fill in your GitHub App ID, private key path, webhook secret, Anthropic API key, and database credentials. All values can be overridden with `KITE_` prefixed environment variables.

### 3. Run migrations

```bash
kite migrate
```

### 4. Start

```bash
kite serve
```

Kite starts an HTTP server (default port 8080) and a worker pool that processes review jobs.

---

## Usage

```bash
# Start the server + workers
kite serve

# Run database migrations
kite migrate

# Check connectivity to MySQL, Valkey, and GitHub
kite health

# Review a PR from the CLI
kite review owner/repo#123
kite review owner/repo#123 --deep
kite review owner/repo#123 --install-id 12345

# Version
kite version
```

### PR Commands

Comment on any PR to trigger Kite:

| Command | Description |
|---------|-------------|
| `/kite review` | Standard review (Claude Sonnet) |
| `/kite deep-review` | Deep review with diagrams (Claude Opus) |
| `/kite ignore` | Skip all future reviews for this PR |

PRs are also reviewed automatically when opened.

---

## How It Works

```
GitHub webhook ──> POST /webhook ──> Valkey queue ──> Worker pool
                   (signature check)   (dedup)        │
                                                      ├── Fetch PR diff (GitHub API)
                                                      ├── Load .kite/ context files
                                                      ├── Split large diffs by token budget
                                                      ├── Call Claude API (per group)
                                                      ├── Validate line numbers against diff
                                                      ├── Post review (summary + inline comments)
                                                      └── Save record to MySQL
```

- **Large PRs** are automatically split into groups that fit within the token budget
- **Inline comments** are validated against real diff ranges — invalid line numbers are dropped
- **Retries** with exponential backoff (3 attempts) — failed jobs go to dead letter queue
- **Webhook dedup** prevents duplicate reviews from redelivered webhooks

---

## Context Files

Create a `.kite/` directory in your repository root with markdown files describing your project's patterns, conventions, and decisions. Kite loads these automatically and includes them in the review prompt.

```
.kite/
  architecture.md    # system design, package structure
  conventions.md     # naming, error handling, patterns
  decisions.md       # ADRs, tech choices
```

Context is capped at ~50K tokens. Files are loaded alphabetically. Subdirectories are supported.

---

## Config Reference

```yaml
github:
  app_id: 12345                          # GitHub App ID
  private_key_path: /etc/kite/app.pem    # Path to private key
  webhook_secret: "secret"               # Webhook secret

llm:
  api_key: "sk-ant-..."                  # Anthropic API key
  review_model: "claude-sonnet-4-6"              # Standard review model
  deep_review_model: "claude-opus-4-6"           # Deep review model

mysql:
  host: localhost                        # MySQL host
  port: 3306                             # MySQL port (default: 3306)
  database: kite                         # Database name
  user: kite                             # Database user
  password: "password"                   # Database password

valkey:
  host: localhost                        # Valkey/Redis host
  port: 6379                             # Valkey/Redis port (default: 6379)

server:
  port: 8080                             # HTTP port (default: 8080)

worker:
  count: 3                               # Concurrent workers (default: 3)

log:
  level: info                            # debug | info | warn | error
```

Every field can be overridden with environment variables using the `KITE_` prefix:

| Variable | Config field |
|----------|-------------|
| `KITE_GITHUB_APP_ID` | `github.app_id` |
| `KITE_GITHUB_PRIVATE_KEY_PATH` | `github.private_key_path` |
| `KITE_GITHUB_WEBHOOK_SECRET` | `github.webhook_secret` |
| `KITE_LLM_API_KEY` | `llm.api_key` |
| `KITE_LLM_REVIEW_MODEL` | `llm.review_model` |
| `KITE_LLM_DEEP_REVIEW_MODEL` | `llm.deep_review_model` |
| `KITE_MYSQL_HOST` | `mysql.host` |
| `KITE_MYSQL_PORT` | `mysql.port` |
| `KITE_MYSQL_DATABASE` | `mysql.database` |
| `KITE_MYSQL_USER` | `mysql.user` |
| `KITE_MYSQL_PASSWORD` | `mysql.password` |
| `KITE_VALKEY_HOST` | `valkey.host` |
| `KITE_VALKEY_PORT` | `valkey.port` |
| `KITE_SERVER_PORT` | `server.port` |
| `KITE_WORKER_COUNT` | `worker.count` |
| `KITE_LOG_LEVEL` | `log.level` |

---

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/webhook` | GitHub webhook receiver |
| `GET` | `/health` | Health check (MySQL + Valkey status) |
| `GET` | `/metrics` | Prometheus metrics |

---

## Stack

| Component | Technology |
|-----------|-----------|
| Language | Go 1.26 |
| Database | MySQL 8.0+ |
| Queue | Valkey 7.0+ (Redis-compatible) |
| LLM | Claude via Anthropic SDK |
| GitHub | go-github v72 + ghinstallation v2 |
| CLI | Cobra |
| Logging | log/slog (JSON structured) |
| Metrics | Prometheus client_golang |
| Migrations | golang-migrate (embedded SQL) |

---

## Build

```bash
make build              # current platform
make release            # cross-compile for linux/darwin amd64/arm64
make test               # run tests
make clean              # remove binaries
```

Binaries are output to `dist/` with SHA256 checksums.

---

## License

MIT — see [LICENSE](LICENSE).
