# Roadmap (Backend)

Each milestone gets its own plan file in `docs/plans/active/` when started. A milestone is done only when `make check` passes and its acceptance criteria are met.

## M0 — Harness and scaffolding
- Repo layout, Makefile, docker compose (api + postgres), Dockerfile, CI running `make check`
- Gin skeleton, config from env, structured logging, `/healthz`, `/readyz`
- golang-migrate and sqlc wired up with one example migration/query
- Accept: `make up` serves health endpoints; CI is green
- Plan: `active/M0-scaffolding.md`

## M1 — Tenancy and auth
- `docs/DATA-MODEL.md`, ADR on tenancy/RLS
- Bank sampah registration, password login for Pengelola, tenant context middleware, token + refresh handling (see ADR 0002)
- Tenant-isolation test harness reusable by later modules
- Accept: tenant A cannot read or write tenant B data, proven by test; login rate limiting works

## M2 — Nasabah and master data
- Nasabah CRUD by Pengelola, password login for Nasabah, QR token generation, optional QR + PIN login, jenis sampah and harga (price history)
- Accept: QR token alone cannot authenticate; price changes do not alter past deposit values

## M3 — Deposits and balance
- Deposit creation with idempotency key, rupiah value and balance calculation, receipt data (struk digital), history, QR lookup of nasabah by Pengelola
- Offline sync semantics documented and tested (replay, out-of-order, duplicates)
- Accept: replayed deposit creates exactly one record; balance always equals the sum of its entries

## M4 — Carbon Ledger
- Write `docs/CARBON-METHOD.md` first (WARM v16 factors, formula, sources, assumptions, limits)
- Versioned emission-factor table, append-only ledger, per-nasabah carbon savings, reversal entries
- Accept: hand-verified calculation test cases; every ledger row cites factor version and source

## M5 — Cashbook, dashboard, reports
- Automatic buku kas, monthly dashboard aggregates, CSV and PDF export
- Accept: cashbook totals reconcile with deposits and sales to pengepul

## M6 — Net Impact Meter and public method page
- Store digital-footprint measurements (SWDM v4), compute net impact, public "Metode & Sumber" data endpoint
- Resolve open question: where CO2.js runs and how the API receives measurements
- Accept: net impact = avoided CO2e minus digital footprint, reproducible from stored inputs

## M7 — Deploy and portability
- Cloud Run + Cloud SQL deployment, Secret Manager, separate migration step
- Verify Supabase Postgres fallback end-to-end (pooler settings, migrations, test suite)
- Accept: same image runs against both databases by changing `DATABASE_URL` only

## Open questions
- Pengelola account recovery path (email-based reset), to settle in M1 (see ADR 0002)
- Where digital-footprint measurement happens (frontend vs backend)
- Emission unit precision (gram vs milligram storage)
- Monorepo vs separate repos for BE and FE
