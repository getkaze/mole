# Code Conventions

## Go Style

- Go 1.26, standard library preferred over third-party when reasonable
- `log/slog` for all logging — structured, JSON output in production
- Errors wrapped with context: `fmt.Errorf("doing X: %w", err)`
- No global state — dependencies injected via constructors (`NewService(...)`)
- No `init()` functions

## Naming

- Packages: short, lowercase, singular (`review`, `queue`, `store`)
- Interfaces: verb-based or role-based (`Provider`, `Store`)
- Constructors: `New<Type>(deps...) *Type`
- Config structs: `<Component>Config` (e.g., `MySQLConfig`, `LLMConfig`)

## Testing

- Table-driven tests preferred
- Test files: `*_test.go` in same package
- No mocks for the database — integration tests hit real MySQL
- Test helpers use `t.Helper()`

## Error Handling

- Return errors, don't panic
- Wrap with context at each layer
- Log at the boundary (handler/worker), not deep in business logic
- `slog.Warn` for recoverable issues, `slog.Error` for failures

## Configuration

- YAML config file (`kite.yaml`) for server settings
- Environment variable overrides with `KITE_` prefix
- Per-repo settings via `.kite/config.yaml`
- Per-repo LLM context via `.kite/*.md`
