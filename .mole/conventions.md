# conventions.md

## Language & Build
- Go (version from `go.mod`, CGO disabled in release builds)
- `go test ./... -race -count=1` is the standard test command
- Binary named `mole`, built from `./cmd/mole`
- `LDFLAGS` inject `main.version` at build time

## Naming Patterns
- **Packages**: short lowercase, single word (`aggregator`, `dashboard`, `github`, `llm`, `store`)
- **Files**: lowercase snake_case matching their primary concern (`client.go`, `class_diagram.go`, `repoconfig.go`)
- **Types**: PascalCase structs and interfaces (`ClientFactory`, `ReviewResponse`, `RouteRegistrar`)
- **Constructors**: `New<Type>(...)` pattern (`NewClaude`, `NewPool`, `NewClientFactory`)
- **Interfaces**: named by capability noun, not prefixed with `I` (`store.Store`, `llm.Provider`)
- **Internal package alias on collision**: `ghclient "github.com/getkaze/mole/internal/github"`, `molestore "github.com/getkaze/mole/internal/store"`
- **Constants**: SCREAMING_SNAKE for exported (`LangEN`, `LangPT`), camelCase for unexported (`sessionCookieName`, `maxContextBytes`)

## Error Handling
- Errors wrapped with `fmt.Errorf("context: %w", err)` throughout
- Functions return `(value, error)` — no panic-based error propagation in application code
- `slog.Error(...)` for non-fatal logged errors (e.g., in aggregator loops); continue processing
- Config validation collects all missing fields into a slice and returns a single combined error
- HTTP handlers use `http.Error(w, msg, code)` directly; no custom error middleware observed

## Configuration
- Primary config: YAML file (`mole.yaml`) loaded via `config.Load(path)`
- Env vars override YAML fields using `MOLE_<SECTION>_<FIELD>` naming (e.g., `MOLE_MYSQL_HOST`)
- Defaults applied before env overrides in `applyDefaults()` then `applyEnvOverrides()`
- Per-repo config in `.mole/config.yaml` fetched from GitHub at review time

## Logging
- Standard library `log/slog` throughout — no third-party logger
- Structured key-value pairs: `slog.Info("msg", "key", value)`
- Log level set from config, parsed in `setupLogging()`

## Testing
- Standard `testing` package, no assertion library observed
- Race detector enabled in CI (`-race`)
- Test files excluded from AST/arch analysis (`strings.HasSuffix(path, "_test.go")`)

## HTTP / Dashboard
- Routes registered with Go 1.22+ method+path syntax: `"GET /path/{param}"`
- Path parameters via `r.PathValue("name")`
- HTMX detected via `r.Header.Get("HX-Request") == "true"`
- Templates: `embed.FS` with `html/template`, layout + page + fragment separation
- Session: HMAC-signed JSON cookie (no third-party session library)

## Module Path
- `github.com/getkaze/mole`
