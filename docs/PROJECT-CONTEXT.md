# Project Context — Setor Hijau

Read this first. It is written so any AI agent or new contributor can understand the project without the original chat history.

## What this is
**Setor Hijau** is a web-based, multi-tenant SaaS information system for waste banks (bank sampah) at RW (Rukun Warga, neighborhood) level in Indonesia. One platform serves many bank sampah; each bank sampah is a tenant whose data is isolated from the others.

Besides recording deposits (setoran) and rupiah balances, it keeps a **Carbon Ledger** (estimated CO2e avoided per deposit) and a **Net Impact Meter** (avoided CO2e minus the app's own estimated digital carbon footprint), so residents see environmental impact, not only money.

## Why it exists
- Course final project: "Proyek Akhir Sistem Informasi Berkelanjutan", Sistem Informasi, Fakultas Ilmu Komputer, Universitas Brawijaya (2026). The written project document (BAB I, in Indonesian) defines the problem, goals, and scope. The docs here are the engineering translation of it.
- Also a portfolio project for the owner (Audy), so quality, tests, and documented decisions matter.
- Sustainability angle: Triple Bottom Line (people, planet, profit) and Green Software Engineering.

## Who
- Owner/lead developer: Audy (Android developer by background; Indonesian speaker). Teammates exist on the course project; this repo is the backend.
- Users: **Nasabah** (RW resident who deposits sorted waste and tracks balance and carbon savings) and **Pengelola** (combined officer and administrator of the bank sampah; at RW scale one person usually does both).
- Only these two roles exist.

## Scope
In scope: bank sampah registration, nasabah management, material types and prices, deposit recording (works offline in the frontend, syncs later), rupiah balance, automatic cashbook, monthly dashboard, CSV/PDF report export, Carbon Ledger, Net Impact Meter, public "Metode & Sumber" (methods and sources) page.

Out of scope: digital payments, marketplace integration, kelurahan dashboard, pickup scheduling, leaderboard. Do not add them. Details: `PRODUCT.md`.

## Tech stack and decisions
| Area | Choice |
|---|---|
| Backend | Go, Gin, pgx v5 + sqlc, golang-migrate |
| Database | PostgreSQL 16.3 |
| Frontend | Next.js (separate directory/repo, not this repo) |
| Local dev | Docker / Docker Compose |
| Primary hosting | Google Cloud Run + Cloud SQL (owner has Google Cloud credit) |
| Fallback | Supabase as Postgres only; switch by changing `DATABASE_URL` |
| Auth | Regular login (identifier + password) primary; QR + PIN optional extra for Nasabah |
| Architecture | Modular monolith, handler -> service -> repository, shared-schema multi-tenancy |

Rationale and alternatives are in `decisions/0001-stack-and-hosting.md` and `decisions/0002-authentication.md`.

## Non-negotiables (summary; full list in `/AGENTS.md`)
- Every tenant-scoped query filters by `bank_sampah_id` taken from the auth context.
- Ledger and deposit records are append-only; corrections are reversing entries.
- No floats for money, weight, or emissions.
- Emission factors live in a versioned table with source; never hardcoded.
- Deposit creation is idempotent.
- Carbon numbers are estimates and must be labeled as such.
- Config via env vars only; DB only via `DATABASE_URL`; migrations plain SQL and run separately.

## Carbon method in one paragraph
Avoided CO2e per deposit = weight x emission factor for that material, using factors from the US EPA Waste Reduction Model (WARM) v16, limited to inorganic recyclable waste diverted from landfill. WARM factors reflect US conditions, so results are estimates. The app's own digital footprint is estimated with the Sustainable Web Design Model v4 (via CO2.js >= 0.18) from data transferred per visit, not measured electricity. Full method to be written in `CARBON-METHOD.md` before milestone M4.

## Current status
_Update this section whenever work is handed off._
- Milestone M0 (scaffolding & harness) is completed:
  - Layout (`cmd/api`, `cmd/migrate`, `internal/platform`, `db/migrations`, `db/queries`).
  - Config from env only with validation, structured JSON logging with `log/slog` (ADR 0003), HTTP middlewares (request ID, structured logger).
  - Health (`/healthz`) and readiness (`/readyz`) endpoints.
  - Database pool via `pgxpool`, migrations runner CLI (`cmd/migrate`) with baseline migration, `sqlc` configured with generated code.
  - Multi-stage `Dockerfile`, `docker-compose.yml`, `Makefile`, `.golangci.yml`, GitHub Actions CI.
  - Tests passing: unit tests for config, logger, and router.
- Milestone M1 (tenancy and auth) is completed:
  - Data model defined in `docs/DATA-MODEL.md` with integer precision (IDR, grams, grams CO2e).
  - ADR 0004 (Application-Enforced Multi-tenancy) and ADR 0005 (Pengelola Account Recovery).
  - Database migration `000002_m1_tenancy_auth` and sqlc query generation for tenants and auth.
  - Tenant registration module (`internal/tenant`) with atomic bank sampah + initial pengelola creation.
  - Auth module (`internal/auth`) with Argon2id password hashing, short-lived JWT, rotating refresh tokens stored hashed in DB, logout, and token-based password reset.
  - Security middlewares (`internal/platform/middleware`): `AuthMiddleware`, `RequireRole`, and in-memory `RateLimiter`.
  - Integration test harness and test suites in `test/integration/` verifying tenant isolation, token rotation, rate limiting, and password reset.
- Next: M2 (Nasabah and master data), see `docs/plans/ROADMAP.md`.

## Open questions
- Where CO2.js runs (frontend most likely) and how the API receives digital-footprint measurements: decide in M6.
- Monorepo vs separate repos for BE and FE.
- Exact WARM material list and mapping to Indonesian waste categories: `CARBON-METHOD.md`.

## Where to look
| Need | File |
|---|---|
| Rules and commands | `/AGENTS.md` |
| Features and roles | `PRODUCT.md` |
| Terms and business invariants | `DOMAIN.md` |
| Structure, tenancy, auth, deploy | `ARCHITECTURE.md` |
| Why a choice was made | `decisions/` |
| What to build next | `plans/ROADMAP.md`, `plans/active/` |

## Handoff protocol for agents
1. Read `/AGENTS.md`, this file, then the plan in `plans/active/`.
2. Do one plan at a time; keep `make check` green.
3. Record non-obvious choices as ADRs, not chat messages.
4. Before stopping, update the plan's status and the "Current status" section above.
5. If something here contradicts the code, fix whichever is wrong and say so in the change.
