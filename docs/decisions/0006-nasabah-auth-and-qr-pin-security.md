# ADR 0006: Nasabah Authentication, QR + PIN Security, and Price History Modeling

Status: Accepted, 2026-11-02

## Context
Milestone 2 requires implementing Nasabah management and Master Data (Jenis Sampah & Harga).
Key product invariants and constraints from `AGENTS.md`, `PRODUCT.md`, and ADR 0002 include:
1. Nasabah login has regular identifier + password as primary, with QR + PIN login as an optional additional method.
2. A QR token alone must **never** grant authentication access.
3. Nasabah usernames are unique within their bank sampah (`bank_sampah_id, username`), not globally across the platform.
4. Pengelola must be able to scan a nasabah QR code to quickly look up the nasabah when recording a deposit, without creating a login session for the nasabah.
5. Material prices change over time, but past deposits must preserve the price in effect when the deposit was recorded. Floating-point numbers are prohibited (Hard Rule #3).

## Decision

### 1. QR Code Opaque Token & Storage
- When a nasabah profile is created, the system generates a cryptographically secure random 32-byte token (hex or base64url encoded).
- The raw QR token is provided only upon initial registration or explicit regeneration by the Pengelola.
- The database stores only the SHA-256 hash of the token (`qr_token_hash`) in `nasabah_profiles`.
- If a card is lost or compromised, the Pengelola can regenerate the QR token, which creates a new token and overwrites `qr_token_hash`, immediately invalidating the old physical card.

### 2. PIN Storage and Brute-Force Lockout
- Nasabah PIN consists of 6 numeric digits (`000000` to `999999`).
- PINs are hashed using Argon2id (`pin_hash`), identical to password security standards.
- To prevent brute-force attacks on the 6-digit space:
  - Each failed attempt increments `failed_pin_attempts`.
  - After 5 consecutive failed attempts, `locked_until` is set to `CURRENT_TIMESTAMP + INTERVAL '15 minutes'`.
  - Successful PIN verification resets `failed_pin_attempts` to 0 and clears `locked_until`.
  - A request providing only a QR token without a PIN, or with an invalid PIN, is rejected with HTTP 401 Unauthorized.

### 3. Nasabah Identifier Resolution for Password Login
- Pengelola usernames and emails are unique platform-wide.
- Nasabah usernames are scoped to their `bank_sampah_id`.
- The `POST /api/v1/auth/login` endpoint accepts an optional `bank_sampah_code` (slug):
  - If `bank_sampah_code` is provided, user lookup searches within that specific tenant.
  - If `bank_sampah_code` is omitted, the system checks globally by email or Pengelola username.

### 4. Pengelola QR Lookup for Deposits
- Endpoint `POST /api/v1/nasabah/lookup-qr` allows an authenticated Pengelola to look up a nasabah by QR token.
- The query strictly enforces `WHERE bank_sampah_id = context.bank_sampah_id AND qr_token_hash = sha256(token)`.
- If a Pengelola scans a QR belonging to another bank sampah, it returns HTTP 404 Not Found (Hard Rule #1 - Tenant Isolation).
- This lookup returns profile details (name, username, RT/RW, active status) and creates **no** authentication session.

### 5. Material Types & Append-Only Price History
- `material_types` defines waste types (e.g., Kardus, Botol Plastik PET) with unique names per tenant.
- `material_prices` stores the price per kilogram in integer Rupiah (`price_per_kg_idr BIGINT`, Hard Rule #3).
- When a price is updated, the previous active price row has its `effective_to` set to `CURRENT_TIMESTAMP`, and a new price row is inserted with `effective_from = CURRENT_TIMESTAMP` and `effective_to = NULL`.
- Current price queries search for `effective_to IS NULL`.
- Past deposits record the price snapshot at transaction time, ensuring historical deposits are immutable.

## Consequences
- Strict compliance with Hard Rule #1 (tenant-isolated lookup), Hard Rule #3 (integer IDR), and Hard Rule #10 (regular login primary, QR+PIN optional extra).
- High security for offline QR cards with hashed tokens and PIN lockout.
