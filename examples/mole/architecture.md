# Architecture

This is a Go monolith following clean architecture.

## Layers
- `cmd/` — Entry points (CLI, server)
- `internal/` — Business logic, not exported
- `pkg/` — Shared libraries, exported

## Patterns
- Repository pattern for database access
- Dependency injection via constructors
- Errors are wrapped with context using `fmt.Errorf("...: %w", err)`

## Code Style
- Use `slog` for structured logging
- Table-driven tests preferred
- No global state — pass dependencies explicitly
