# ADR 0004: Multi-tenancy Isolation Strategy

Status: Accepted, 2026-11-01

## Context
Setor Hijau is a multi-tenant SaaS waste bank system where each bank sampah is an isolated tenant.
In `docs/ARCHITECTURE.md` and `docs/PROJECT-CONTEXT.md`, defense-in-depth via PostgreSQL Row-Level Security (RLS) driven by `SET LOCAL app.bank_sampah_id = ...` was considered alongside application-level filtering.

However, Setor Hijau supports Supabase / PgBouncer as a database fallback (see ADR 0001). In transaction-pooled or connection-pooled architectures (such as Supavisor or PgBouncer in transaction mode):
1. Session-level variables (`SET`) leak across pooled connections if executed outside an explicit transaction block (`BEGIN ... COMMIT`).
2. Enforcing `SET LOCAL` requires every single read query to open an explicit database transaction, incurring roundtrips, transaction overhead, and complexity.
3. Our stack uses `sqlc` with `pgx/v5`, which compiles strictly typed SQL queries where query parameters are explicit and verifiable at compile time.

## Decision
1. **Application-Enforced Tenancy as the Primary Layer**:
   - Every database query for tenant-scoped tables must explicitly include `WHERE bank_sampah_id = $1` (Hard Rule #1).
   - The `bank_sampah_id` is always resolved by the authenticated JWT context in middleware (`internal/platform/middleware`), never accepted from user request bodies, path parameters, or query strings.
   - Handlers pass the tenant ID extracted from `context.Context` to the service and repository layers.
2. **Database Integrity & Constraints**:
   - Foreign keys to `bank_sampah(id)` with `ON DELETE CASCADE` or `RESTRICT`.
   - Composite unique constraints including `bank_sampah_id` (e.g., `UNIQUE (bank_sampah_id, username)` for nasabah accounts, `UNIQUE (bank_sampah_id, name)` for waste categories).
   - Composite indexes on `(bank_sampah_id, id)` or `(bank_sampah_id, created_at)` for query performance and tenant isolation.
3. **Automated Verification Harness**:
   - A reusable integration test suite (`test/integration/tenant_isolation_test.go`) explicitly tests cross-tenant access for all tenant-scoped resources: Tenant A's token cannot read, update, or delete Tenant B's records.

## Alternatives Considered
- **PostgreSQL RLS with `SET LOCAL`**: Rejected as mandatory default because of transaction-pooler constraints and unnecessary transaction overhead for simple SELECT queries. May be reconsidered as an optional migration if direct Cloud SQL connections without transaction pooling are guaranteed.
- **Separate Schemas per Tenant**: Rejected due to high migration complexity across dozens or hundreds of RW bank sampah, and difficulty in aggregating anonymized sustainability metrics (Net Impact Meter).

## Consequences
- Developers and agents must maintain `bank_sampah_id` filters in all sqlc queries touching tenant tables.
- CI and integration tests enforce tenant boundary checks on every PR or commit.
- High connection pooler compatibility: works transparently with Cloud SQL direct connections, PgBouncer, and Supabase poolers without session leaks.
