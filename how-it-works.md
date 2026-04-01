# How It Works — PR Review Flow

Complete step-by-step trace of what happens when someone opens a PR in a repository with Mole installed.

---

## 1. Webhook Reception

GitHub sends a `POST /webhook`. The handler:
- Reads the body (up to 10MB)
- Validates HMAC-SHA256 signature via `X-Hub-Signature-256`
- Extracts the `X-GitHub-Delivery` (unique event ID)

## 2. Deduplication

Checks Valkey if the delivery was already processed:
```
EXISTS mole:dedup:{deliveryID}
```
If it exists → returns 200 and stops. Otherwise, continues.

## 3. Job Creation

Parses the `pull_request` event, checks if action is `"opened"`, and creates the job:
```json
{
  "id": "pr-owner/repo-123-1711792800000",
  "type": "deep",
  "repo": "owner/repo",
  "pr_number": 123,
  "install_id": 987654321,
  "delivery_id": "abc-123-..."
}
```

## 4. Enqueue to Valkey

```
LPUSH mole:queue:jobs <job JSON>
```

Marks the delivery as processed (72h TTL):
```
SET mole:dedup:{deliveryID} "1" EX 259200
```

## 5. Worker Consumes

The worker pool continuously does a blocking pop:
```
BRPOP mole:queue:jobs 5
```
LPUSH + BRPOP = FIFO. The worker deserializes the JSON back into a `Job` struct.

## 6. Retry with Backoff

If processing fails, retries up to 3 times with exponential backoff (1s, 2s, 4s). If exhausted:
```
LPUSH mole:queue:deadletter <job JSON>
```

## 7. Review Execution — `Service.Execute()`

The review service runs this sequence:

### 7a. Check if PR is ignored
```sql
SELECT COUNT(*) FROM ignored_prs WHERE repo = ? AND pr_number = ?
```

### 7b. "Eyes" reaction (visual signal that processing started)
```
POST /repos/{owner}/{repo}/issues/{pr}/reactions → {"content": "eyes"}
```

### 7c. Fetch PR metadata
```
GET /repos/{owner}/{repo}/pulls/{pr}
```
Extracts: `head SHA`, `base ref`, `author login`.

### 7d. Fetch paginated diff
```
GET /repos/{owner}/{repo}/pulls/{pr}/files?per_page=100&page=1
GET /repos/{owner}/{repo}/pulls/{pr}/files?per_page=100&page=2
...
```
Each file comes with `filename`, `status`, `patch` (unified diff).

### 7e. Load `.mole/` context
```
GET /repos/{owner}/{repo}/contents/.mole?ref={base_ref}
GET /repos/{owner}/{repo}/contents/.mole/architecture.md?ref={base_ref}
GET /repos/{owner}/{repo}/contents/.mole/conventions.md?ref={base_ref}
...
```
Concatenates all `.md` files up to 200KB. If `.mole/` doesn't exist, continues without context.

### 7f. Load repo config
```
GET /repos/{owner}/{repo}/contents/.mole/config.yaml?ref={base_ref}
```
Defines: personality, min_severity, ignore patterns, custom instructions.

### 7g. Exploration Stage (two-stage pipeline)

If `repos.base_path` is configured in `mole.yaml` and `git` is available on the host, Mole runs a codebase exploration before calling the review model.

#### Clone / Fetch

The RepoManager (`internal/git/repo.go`) ensures the repository exists locally:
- **First time:** bare clones the repo to `{base_path}/{owner}/{repo}`. Posts a PR comment: "Cloning repository for the first time..."
- **Subsequent:** runs `git fetch --all --prune` to update

Authentication uses a GitHub App installation token passed via a git credential helper (environment variables — token never appears in CLI args or config files).

A per-repo `sync.Mutex` serializes git operations (clone/fetch/worktree add/remove) so concurrent PRs on the same repo don't corrupt git state. The lock only covers git operations — the actual review runs outside the lock, in parallel.

#### Worktree

Creates an isolated worktree for the PR branch:
```
git -C {base_path}/{owner}/{repo} worktree add /tmp/mole-wt-XXXXX origin/{branch}
```
Cleanup is deferred — the worktree directory is removed regardless of success or failure.

#### Haiku Exploration (Stage 1)

The Explorer (`internal/llm/explorer.go`) runs a multi-turn tool use conversation with Claude Haiku:

1. **System prompt:** embedded markdown agent instructions (`internal/llm/agents/explorer.md`) via `go:embed`
2. **User message:** Go-generated file tree + PR diff
3. **Tools available:**
   - `get_file(path)` — reads a file (max 100KB, path-validated)
   - `search_code(query, file_pattern?)` — regex search across files (max 50 matches)
   - `list_dir(path)` — lists a directory's contents
4. **Loop:** Haiku calls tools → Go executes them in the worktree → results sent back → repeat until Haiku is done or max turns reached

All tools are sandboxed to the worktree directory via `safePath()` — resolves the absolute path, verifies it starts with the worktree root, and checks for symlink escape.

**Configuration** (in `mole.yaml`):
```yaml
repos:
  base_path: "/var/lib/mole/repos"
exploration:
  max_turns: 25
  model: "claude-haiku-4-5-20251001"
```

The exploration uses non-streaming API calls (needs the full response to detect tool_use blocks before continuing).

#### Opus/Sonnet Review (Stage 2)

The collected context from Haiku is appended to the existing `.mole/` context and passed to the review model. The review call itself is unchanged — it just receives richer context.

#### Graceful Fallback

If anything fails (no base_path configured, git not installed, clone fails, exploration errors), the review falls back to the current diff-only behavior with a WARNING log. The pipeline never crashes due to exploration failures.

### 7h. Load previous issues from the same PR (to avoid duplicates)
```sql
SELECT ... FROM issues i JOIN reviews r ON r.id = i.review_id
WHERE r.repo = ? AND r.pr_number = ? ORDER BY i.created_at
```
Formatted as text so the LLM knows what was already reported.

## 8. Model Selection

> **Note:** Regardless of model selection below, both standard and deep reviews run the Haiku exploration stage (when enabled). The model selection only affects the final review call.

```go
model = s.opus   // job.Type == "deep" → Claude Opus
model = s.sonnet // job.Type == "standard" → Claude Sonnet
```

## 9. Claude API Call (Streaming)

The prompt is built with `BuildPrompt()` — the system prompt activates 5 internal "agents": Security Sentinel, Bug Hunter, Architect, Performance Analyst, Code Quality Reviewer. Deep review instructs more exhaustive analysis.

The user prompt concatenates:
1. Project context (`.mole/*.md`)
2. Repo instructions (`config.yaml`)
3. Previously reported issues on the PR
4. Diff with added line numbers

```
POST https://api.anthropic.com/v1/messages (streaming)
{
  "model": "claude-opus-4-6",
  "max_tokens": 128000,
  "system": [{"type": "text", "text": "<system prompt with 5 agents>"}],
  "messages": [{"role": "user", "content": "<context + numbered diff>"}]
}
```

## 10. Response Parsing

Claude returns JSON with: `summary`, `comments[]`, `suggestions[]`, `diagrams[]`. The parser extracts and maps them to structs.

## 11. Comment Validation

`ValidateComments()` cross-references each comment with the diff — only keeps those pointing to **addition lines** (lines with `+`). Comments on context lines or files outside the diff are discarded.

## 12. Repo Filters

`FilterComments()` applies:
- `min_severity` — e.g. if `attention`, drops all `suggestion` items
- `ignore` patterns — e.g. `**/*_test.go`, `vendor/**`
- `max_inline_comments` — caps with priority by severity

## 13. Score Calculation

```
100 - (critical × 5) - (attention × 2) - (suggestion × 1)
```

## 14. Personality Formatting

The formatter applies the configured personality (`mole`, `formal`, `minimal`) and generates:
- Review body (summary + score badge + issues list + suggestions + mermaid diagrams)
- Inline comments formatted with severity emoji

## 15. Post to GitHub

```
POST /repos/{owner}/{repo}/pulls/{pr}/reviews
{
  "commit_id": "{headSHA}",
  "body": "## 🔍 Code Review\n\n🟡 Score: 78/100\n\n...",
  "event": "COMMENT",
  "comments": [
    {"path": "src/service.go", "line": 42, "body": "🔴 **Critical**...", "side": "RIGHT"},
    ...
  ]
}
```

Then fetches the comment IDs:
```
GET /repos/{owner}/{repo}/pulls/{pr}/reviews/{review_id}/comments
```

## 16. Success Reaction

```
POST /repos/{owner}/{repo}/issues/{pr}/reactions → {"content": "rocket"}
```

## 17. Save to MySQL

**Review:**
```sql
INSERT INTO reviews (repo, pr_number, pr_author, review_type, model, score,
  input_tokens, output_tokens, status, summary, error_message, installation_id)
VALUES ('owner/repo', 123, 'alice', 'deep', 'claude-opus-4-6', 78,
  4523, 1247, 'success', 'Good PR...', NULL, 987654321)
```

**Issues (in a transaction):**
```sql
INSERT INTO issues (review_id, pr_author, category, subcategory, severity,
  file_path, line_number, description, suggestion, module_name)
VALUES (456, 'alice', 'Security', 'XSS', 'critical',
  'src/service.go', 42, 'User input not escaped...', NULL, 'src')
```

**Link comment IDs (for reaction tracking):**
```sql
UPDATE issues SET github_comment_id = 9876543210 WHERE id = 789
```

## 18. Prometheus Metrics

Records review duration and tokens used for observability.

---

## Flow Diagram

```
GitHub POST /webhook
  → Signature check (HMAC-SHA256)
  → Dedup check (EXISTS in Valkey)
  → LPUSH mole:queue:jobs
  → SET mole:dedup:{id} (72h TTL)
        ↓
Worker BRPOP mole:queue:jobs
  → Check ignored PR (MySQL)
  → 👀 Reaction
  → Fetch PR info + diff (GitHub API)
  → Load .mole/ context + config (GitHub API)
  → [if enabled] Clone/fetch repo + create worktree
  → [if enabled] Haiku explores codebase via tools (multi-turn)
  → Load previous issues (MySQL)
  → Build prompt (diff + .mole/ context + exploration context)
  → Call Claude for review (streaming)
  → Parse JSON → Validate lines → Filter → Score
  → Format with personality
  → POST review + inline comments (GitHub API)
  → 🚀 Reaction
  → Save review + issues (MySQL)
  → Link comment IDs (MySQL)
  → [defer] Remove worktree
```

## Key Files Reference

| Component | File | Key Functions |
|-----------|------|---------------|
| Webhook Handler | `internal/server/webhook.go` | `ServeHTTP()`, `handlePullRequest()` |
| Queue (Valkey) | `internal/queue/queue.go` | `Enqueue()` (LPUSH), `Dequeue()` (BRPOP), `IsDuplicate()` (EXISTS), `MarkProcessed()` (SET) |
| Worker Pool | `internal/worker/worker.go` | `run()`, `processWithRetry()` |
| Review Service | `internal/review/service.go` | `Execute()` — orchestrates entire flow |
| GitHub Client | `internal/github/client.go` | `NewClientFactory()`, `Client()` |
| Diff Fetching | `internal/github/diff.go` | `FetchDiff()` |
| Context Loading | `internal/github/context.go` | `LoadContext()` |
| Repo Config | `internal/github/repoconfig.go` | `LoadRepoConfig()` |
| Review Posting | `internal/github/review.go` | `PostReview()`, `GetPRInfo()` |
| LLM Prompt | `internal/llm/prompt.go` | `BuildPrompt()`, `numberDiffLines()` |
| Claude Client | `internal/llm/claude.go` | `Review()` — calls Anthropic API |
| Response Parser | `internal/llm/parser.go` | `ParseResponse()` |
| Validator | `internal/review/validator.go` | `ValidateComments()` — checks only addition lines |
| Filter | `internal/review/filter.go` | `FilterComments()` — severity, patterns, max |
| Formatter | `internal/review/formatter.go` | `Format()` — creates GitHub-ready review |
| Score | `internal/score/score.go` | `Calculate()` |
| MySQL Store | `internal/store/mysql.go` | `SaveReview()`, `SaveIssues()`, `UpdateIssueCommentID()` |
| Repo Manager | `internal/git/repo.go` | `Prepare()` — clone/fetch + worktree, `Cleanup()`, `CleanupStale()` |
| Explorer | `internal/llm/explorer.go` | `Explore()` — Haiku multi-turn tool use loop |
| Exploration Tools | `internal/llm/tools.go` | `Execute()` — get_file, search_code, list_dir (sandboxed) |
| File Tree | `internal/llm/tree.go` | `BuildTree()` — generates directory tree for prompt |
| Explorer Prompt | `internal/llm/agents/explorer.md` | System prompt for Haiku (embedded via go:embed) |
| PR Comments | `internal/github/comment.go` | `PostComment()`, `EditComment()` — clone status feedback |
