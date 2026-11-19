# Milestone M7: Deploy and Portability

Status: **Code & Docs Ready — Awaiting Live GCP + NeonDB Verification**  
Started: 2026-11-15  

## Goal
Prepare `setor-hijau-be` for production deployment on **Google Cloud Run** with **NeonDB (PostgreSQL 16)** as the primary managed database, while preserving full `DATABASE_URL`-only portability across Cloud SQL, Supabase Postgres, and local Docker Postgres.

## Scope
1. **Pooler & Serverless DB Compatibility (`internal/platform/db`)**:
   - Configure `pgxpool` with `pgx.QueryExecModeExec` by default (unless overridden in `DATABASE_URL`) for transaction pooler compatibility (NeonDB PgBouncer & Supabase Supavisor).
   - Set `MinConns = 0` and `MaxConnIdleTime = 5m` so NeonDB compute can auto-suspend.
   - Increase startup/ping timeouts (`10s` ping, `15s` startup) for serverless DB cold wake-up.
   - Support configurable `DB_MAX_CONNS` (default `10`).
2. **CORS Middleware (`internal/platform/middleware`, `internal/platform/config`, `internal/platform/router`)**:
   - Add `CORS_ALLOWED_ORIGINS` env config and zero-dependency `middleware.CORS`.
   - Increase `/readyz` ping timeout to `5s` for serverless DB wake-up tolerance.
3. **Deployment Documentation & ADR**:
   - Add ADR `0011-neondb-primary-database.md`.
   - Add `docs/DEPLOYMENT.md` covering NeonDB Console setup, GCP Console setup (Artifact Registry, Secret Manager, Cloud Run Job for migrations, Cloud Run Service for API), and portability verification.
4. **Verification**:
   - `make check` passes (`go vet`, `go test -v -race ./...`, `sqlc` diff check).
   - Update `docs/PROJECT-CONTEXT.md` and `docs/plans/ROADMAP.md`.
