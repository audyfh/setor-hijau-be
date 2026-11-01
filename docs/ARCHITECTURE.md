# Architecture

## Style
Modular monolith in Go. One deployable, one database, clear module boundaries. Chosen over microservices because the scope is small and the aim is correctness, auditability, and low resource use (consistent with the project's Green Software Engineering goal).

## Layers (per module)
`handler (Gin)` -> `service (business rules)` -> `repository (sqlc-generated queries over pgx)`
- Handlers: parse/validate input, call a service, map errors to HTTP. No business logic, no SQL.
- Services: business rules and transactions. Depend on repository interfaces.
- Repositories: only place that touches the database.
- A module never reads another module's tables; it calls that module's service.

## Modules
auth, tenant (bank sampah), nasabah, material (jenis and harga sampah), deposit (setoran, balance), cashbook (buku kas), carbon (Carbon Ledger, emission factors), report (dashboard, CSV/PDF export), impact (Net Impact Meter, public Metode & Sumber data).

## Multi-tenancy
- Model: shared database, shared schema, `bank_sampah_id` column on every tenant-scoped table.
- Tenant id is resolved from the authenticated token in middleware and placed in the request context. Services/repositories read it from context; it is never taken from request bodies or query strings.
- Defense in depth (planned, confirm via ADR in M1): PostgreSQL row-level security driven by `SET LOCAL app.bank_sampah_id = ...` inside each transaction. `SET LOCAL` also works behind Supabase's transaction pooler.
- Required test: for each tenant-scoped table, data created by tenant A is invisible and unmodifiable by tenant B.

## Auth (see `decisions/0002-authentication.md`)
- Primary, both roles: regular login with an identifier (username) and password. Passwords hashed with argon2id (or bcrypt if argon2id is unavailable), login attempts rate limited.
- Additional, Nasabah only: QR code + PIN login. Optional, enabled per nasabah by setting a PIN. Never the only way in.
- QR code is also an identity token that the Pengelola scans to pick a nasabah when recording a deposit. That is identification, not login, and grants no access by itself.
- Accounts: a Pengelola registers a bank sampah (creating the tenant) with their own credentials. Nasabah accounts are created by the Pengelola, since residents may not have email.
- Short-lived access token carrying user id, role, and bank_sampah_id, plus a rotating refresh token stored hashed server-side. Two roles only: Nasabah, Pengelola.

## Deposits and offline sync
The frontend records deposits offline and syncs later. The API must:
- accept a client-generated idempotency key per deposit and return the original result on replay;
- store the client-recorded timestamp separately from the server-received timestamp;
- compute rupiah value and CO2e in the same transaction using the prices and factor version in effect, and store both on the record.

## Carbon
- Ledger rows store integer CO2e, material, weight, and the factor id + version + source used, so every figure is auditable. Unit precision (gram vs milligram) is decided in `DATA-MODEL.md`.
- Net Impact = total avoided CO2e minus estimated digital footprint (SWDM v4 via CO2.js, based on data transfer). CO2.js is a JavaScript library, so measurement likely happens in the Next.js app; the API stores the reported measurements. Open question for M6.
- Details belong in `docs/CARBON-METHOD.md`.

## Runtime and deployment
- One multi-stage Dockerfile producing a small static binary image; stateless; listens on `$PORT`.
- Local: docker compose (api + postgres).
- Primary: Cloud Run (API) + Cloud SQL for PostgreSQL. Secrets in Secret Manager, injected as env vars. Migrations run as a separate job/step.
- Fallback: Supabase as the Postgres database only. Switching means changing `DATABASE_URL` and, if using the transaction pooler, configuring pgx to avoid prepared-statement caching (e.g. simple protocol or describe-cache mode). Auth stays application-owned; do not depend on Supabase Auth or its client libraries. If Google Cloud becomes unusable, the container also needs another container host; choose via a new ADR at that time.
- Portability rules: stick to standard PostgreSQL features, keep migrations plain SQL, no cloud SDKs outside `internal/platform`.

## Observability
Structured JSON logs to stdout, request id per request, `/healthz` and `/readyz` endpoints. Never log passwords, PINs, tokens, or QR secrets.
