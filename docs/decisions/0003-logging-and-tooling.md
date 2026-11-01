# ADR 0003: Structured Logging and Core Tooling

Status: Accepted, 2026-11-01

## Context
Milestone 0 requires structured JSON logging for observability (request ID, method, path, latency, status code) while avoiding vendor lock-in or unnecessary heavyweight dependencies. Furthermore, passwords, PINs, tokens, and QR secrets must never be logged.

## Decision
1. **Logging Library**: Use Go 1.21+ standard library `log/slog` with `slog.NewJSONHandler(os.Stdout, ...)`.
   - Zero external logging dependencies (lightweight, Green Software Engineering alignment).
   - High performance, type-safe attributes, structured JSON format by default.
   - Enforce redaction of sensitive fields (passwords, PINs, auth tokens) in middlewares and loggers.
2. **Migrations Runner**: Standalone runner in `cmd/migrate/main.go` using `github.com/golang-migrate/migrate/v4`.
   - Decoupled from application startup (Hard Rule #7 in `AGENTS.md`).
   - Plain SQL migration files in `db/migrations/`.
3. **Database Query Generation**: `sqlc` with pgx/v5 engine.
   - Type-safe, compile-time verified queries in `db/queries/`.
   - Avoids ORM magic while ensuring strict tenant isolation query filters.

## Consequences
- No extra dependencies like Zap or Zerolog needed; `log/slog` is standard Go.
- Easy to pipe JSON logs to Cloud Logging in Cloud Run or stdout in Docker Compose.
