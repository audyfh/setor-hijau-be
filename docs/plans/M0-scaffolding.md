# Plan: M0 — Harness and scaffolding

Status: not started
Roadmap: `../ROADMAP.md` (M0)

## Goal
A running skeleton where `make up` serves health endpoints and `make check` is green in CI, so every later change has guardrails.

## Steps
1. Initialize Go module and the layout from `/AGENTS.md` (`cmd/api`, `internal/platform`, `db/`).
2. Config from environment variables only (`DATABASE_URL`, `PORT`, log level). Fail fast on missing required values.
3. Gin server with request id middleware, structured JSON logging, `/healthz` (process up) and `/readyz` (database reachable).
4. Docker: multi-stage Dockerfile (small static binary, listens on `$PORT`) and `docker-compose.yml` (api + postgres 16.3).
5. golang-migrate wired up as a separate command (not on startup); one example migration.
6. sqlc configured with one example query and generated code committed.
7. Makefile targets: `up`, `down`, `test`, `lint`, `sqlc`, `migrate-new`, `migrate-up`, `migrate-down`, `check`.
8. golangci-lint config; integration test helper that starts a real Postgres container.
9. CI workflow running `make check`.
10. Update `docs/PROJECT-CONTEXT.md` "Current status".

## Acceptance
- `make up` then `GET /healthz` and `/readyz` return 200.
- `make check` passes locally and in CI.
- No cloud-vendor SDK imports; no float types for money/weight/emission.

## Notes
- Do not start M1 (tenancy/auth) in this plan.
- Record any non-obvious choice (for example the Go logging library) as an ADR.
