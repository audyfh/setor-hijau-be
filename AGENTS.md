# AGENTS.md — Setor Hijau Backend

Setor Hijau is a multi-tenant SaaS waste bank (bank sampah) system at RW level. Each bank sampah is a tenant.
Besides rupiah balances, it records a per-deposit carbon ledger (CO2e avoided) and a Net Impact Meter
(avoided CO2e minus the app's own digital footprint). This repo is the **backend only**; the frontend is Next.js in a separate directory.

**New here (human or AI agent)? Read `docs/PROJECT-CONTEXT.md` first**, then the doc for the area you are touching.
This file is a map and the rulebook. Details live in `docs/`.

## Stack
- Go (latest stable), Gin (HTTP), pgx v5 + sqlc (database), golang-migrate (plain SQL migrations)
- PostgreSQL 16.3, Docker / Docker Compose for local dev
- Primary hosting: Google Cloud Run + Cloud SQL. Fallback: Supabase as a Postgres-only database. See `docs/decisions/0001-stack-and-hosting.md`.

## Commands (create these Makefile targets in M0; keep them working)
- `make up` / `make down` — local stack (api + postgres) via docker compose
- `make test` — unit + integration tests (integration uses a real Postgres container)
- `make lint` — golangci-lint + `go vet`
- `make sqlc` — regenerate query code after editing `db/queries/*.sql`
- `make migrate-new name=<x>` / `make migrate-up` / `make migrate-down`
- `make check` — lint + test + sqlc diff check; must pass before any change is considered done

## Hard rules (do not violate; tests enforce the starred ones)
1. * Every query on tenant-scoped tables filters by `bank_sampah_id`. The tenant id comes from the auth context, never from request input. See `docs/ARCHITECTURE.md`.
2. * `carbon_ledger` is append-only. Corrections are new reversing entries, never UPDATE/DELETE. Same for deposit records.
3. * No floats for money, weight, or emissions. Use integer rupiah, integer grams, and fixed-precision/integer emission units.
4. Emission factors are never hardcoded in Go. They live in a versioned DB table with source and version, and every ledger row stores the factor version used. Method details: `docs/CARBON-METHOD.md` (to be written before M4).
5. Deposit creation is idempotent (client-supplied idempotency key) because the frontend records offline and syncs later.
6. Config comes only from environment variables. The database is reached only through `DATABASE_URL`, so Cloud SQL and Supabase are interchangeable. No cloud-vendor SDKs in domain/service code.
7. Migrations are plain, portable SQL (no Cloud SQL-specific features) and run as a separate step, never on app startup.
8. Stay in scope: only two roles (Nasabah, Pengelola). Out of scope: digital payments, marketplace integration, kelurahan dashboard, pickup scheduling, leaderboard. See `docs/PRODUCT.md`.
9. Carbon results are estimates (WARM factors are US-based). Never present them as exact measurements in API field names or docs.
10. * Regular login (identifier + password) is the primary authentication for both roles. QR + PIN login is an optional extra for Nasabah and must never be the only way in. Passwords and PINs are stored hashed and never logged. See `docs/decisions/0002-authentication.md`.

## Layout (target)
```
cmd/api/            entrypoint
internal/<module>/  handler.go, service.go, repository.go, model.go (modules: auth, tenant, nasabah, material, deposit, cashbook, carbon, report, impact)
internal/platform/  config, db, http middleware, logging
db/migrations/      SQL migrations
db/queries/         sqlc input
docs/               system of record
```
Dependency direction: handler -> service -> repository. Modules talk to each other through service interfaces, not each other's tables.

## Working agreements
- Work from the active plan in `docs/plans/active/`; one plan file per task, move to `completed/` when done.
- Non-obvious design choice? Add an ADR in `docs/decisions/` instead of leaving it in chat.
- If a rule here or in docs is missing or wrong, propose the doc change in the same change set.
- Definition of done: `make check` passes, new behavior has tests, tenant-isolation test covers any new tenant-scoped table, docs updated.
- Handing off to another agent or session: update the plan file's status and the "Current status" section of `docs/PROJECT-CONTEXT.md` so the next one starts from facts, not chat history.
- User-facing text (API messages shown to end users, docs for the thesis) is in Indonesian; code, identifiers, and engineering docs are in English.

## Docs index
- `docs/PROJECT-CONTEXT.md` — start here: what, why, who, scope, status, open questions
- `docs/PRODUCT.md` — scope, roles, features, out-of-scope
- `docs/ARCHITECTURE.md` — structure, tenancy, auth, deployment
- `docs/DOMAIN.md` — glossary and business invariants
- `docs/DATA-MODEL.md` — schema (to write in M1)
- `docs/CARBON-METHOD.md` — WARM/SWDM formulas and sources (to write before M4)
- `docs/plans/ROADMAP.md` — milestones; `docs/plans/active/` — current task plans
- `docs/decisions/` — ADRs (0001 stack and hosting, 0002 authentication)
