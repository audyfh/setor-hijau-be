# ADR 0007: Deposit Idempotency, Price Snapshotting, and Offline Sync Semantics

## Status
Accepted

## Context
At the RW level in Indonesia, Bank Sampah transactions frequently happen outdoors or in neighborhood outposts (pos RW) with spotty or nonexistent internet connectivity.
The frontend must be able to record deposits locally in an offline storage queue (e.g., IndexedDB) and synchronize transactions when connectivity is restored.
Furthermore:
1. Waste bank physical scales measure in grams (or fractional kilograms), while prices are defined per kilogram (`price_per_kg_idr`).
2. Hard Rule #3 forbids floats for money, weight, and emissions.
3. Hard Rule #5 mandates idempotent deposit creation with a client-supplied idempotency key.
4. Prices change over time; deposits recorded offline at 09:00 must not inherit a price change enacted at 10:00 before sync.
5. Invariant #2 requires that a nasabah's balance always equals the sum of their transactions.

## Decisions

### 1. Separation of Timestamps
Every deposit stores two separate timestamps (Invariant #8):
- `client_recorded_at TIMESTAMPTZ NOT NULL`: The exact time the physical deposit occurred on the client device.
- `created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP`: The server-side receipt and persistence timestamp.

### 2. Historical Price Snapshotting
When calculating the value of each deposit line, the server resolves the material price that was active at `client_recorded_at`:
```sql
SELECT price_per_kg_idr
FROM material_prices
WHERE material_type_id = $1
  AND bank_sampah_id = $2
  AND effective_from <= $3
  AND (effective_to IS NULL OR effective_to > $3)
ORDER BY effective_from DESC
LIMIT 1;
```
The resolved unit price (`price_per_kg_idr`) and the computed line subtotal (`subtotal_idr`) are permanently snapshotted into `deposit_lines`. Future price updates never rewrite or recalculate historical deposit values (Hard Rule #2 & Invariant #4).

### 3. Half-Up Integer Rupiah Rounding
Because weights are recorded in grams and prices are per kilogram (1,000 grams):
$$\text{subtotal\_idr} = \left\lfloor \frac{\text{weight\_grams} \times \text{price\_per\_kg\_idr} + 500}{1000} \right\rfloor$$
This standard half-up integer division rounds to the nearest Indonesian Rupiah without floating-point representation or rounding bias.

### 4. Idempotency Key & Replay Semantics
- Each deposit request includes a client-generated UUIDv4 `idempotency_key`.
- Database uniqueness constraint: `CONSTRAINT uq_deposits_bank_sampah_idempotency UNIQUE (bank_sampah_id, idempotency_key)`.
- **Replay behavior**: If a request with an existing `(bank_sampah_id, idempotency_key)` is received, the server retrieves and returns the existing deposit record with `HTTP 200 OK` (or 201 Created) instead of returning an error.
- This guarantees that repeated submissions from background sync workers or retries after network timeouts do not cause duplicate balance credits.

### 5. Out-of-Order Delivery & Derived Balance
Offline synchronization may deliver transactions out of chronological order (e.g. deposit recorded at 10:15 arrives before deposit recorded at 09:30 due to retry race conditions).
- Because pricing is pinned to `client_recorded_at` and balances are derived dynamically (`SUM(total_amount_idr)`), out-of-order delivery has no effect on financial consistency.
- Balances are never maintained as a mutable column in the database; they are computed directly from ledger/deposit rows.

## Consequences
- **Positive**: Resilient against network drops, safe offline replay, mathematically exact balance derivation, audit trail preservation.
- **Negative**: Client clock drift could theoretically impact price resolution if client time is far in the past; server validates that `client_recorded_at` is not in the future beyond a reasonable threshold (e.g. max 5 minutes ahead of server clock).
