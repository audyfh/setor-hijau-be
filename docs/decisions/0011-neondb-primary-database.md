# ADR 0011: NeonDB as Primary Managed PostgreSQL with Cloud Run

Status: Accepted (project owner), 2026-11-15

## Context
[ADR 0001](0001-stack-and-hosting.md) originally selected Google Cloud SQL as the primary PostgreSQL database and Supabase (Postgres-only) as the fallback. However, Cloud SQL incurs a continuous minimum monthly cost even when the application has no traffic, whereas the backend runs on Cloud Run (which scales to zero) and the project has a Green Software Engineering focus.

## Decision
- Use **NeonDB (PostgreSQL 16)** as the primary managed database alongside **Google Cloud Run** for the Go API container.
- Keep **Cloud SQL** and **Supabase (Postgres-only)** fully supported as drop-in alternatives by changing only `DATABASE_URL`.
- Configure `internal/platform/db`:
  - `pgx.QueryExecModeExec` by default (unless `default_query_exec_mode` is explicitly set in `DATABASE_URL`) so the exact same image works behind NeonDB's PgBouncer pooler and Supabase's Supavisor transaction pooler.
  - `MinConns = 0` and `MaxConnIdleTime = 5m` so idle connections close cleanly and NeonDB compute can scale to zero.
  - `10s` ping timeout (`15s` startup context) to absorb serverless database cold wake-up (~1–2 seconds).
- Use **two** NeonDB connection strings operationally:
  - **Pooled** (`-pooler` hostname) for the Cloud Run API service (`DATABASE_URL`).
  - **Direct / Unpooled** (standard hostname without `-pooler`) for `cmd/migrate` (`DATABASE_URL_DIRECT`) because `golang-migrate` uses PostgreSQL advisory locks.

## Consequences
- Near-zero idle compute cost and automatic scale-to-zero on both API (Cloud Run) and DB (NeonDB), aligning with Green Software Engineering.
- First request after >5 minutes of DB inactivity may incur ~1–2s cold wake-up latency.
- No vendor lock-in: `DATABASE_URL` remains the sole database boundary.
