# ADR 0002: Authentication

Status: Accepted by project owner (direction); details provisional until M1 implementation, 2026-11-01

## Context
The BAB I project document describes Nasabah login via QR code + PIN. The owner decided that regular login must exist, with QR code only as an addition. Residents may not have email addresses. Pengelola need a credential method that works for bank sampah registration.

## Decision
1. **Primary login (both roles):** identifier + password.
   - Identifier is a username unique within the platform for Pengelola (email may be used as the identifier and also serve recovery), and unique within the bank sampah for Nasabah.
   - Passwords hashed with argon2id (bcrypt acceptable if argon2id is unavailable). Minimum length enforced; no composition theatre.
   - Login attempts rate limited per identifier and per IP.
2. **Additional login (Nasabah only):** QR code + PIN.
   - Optional per nasabah: only usable after the nasabah (or Pengelola on their behalf) sets a PIN.
   - The QR carries an opaque, unguessable token (not the nasabah id or a sequential number), revocable and regenerable by the Pengelola.
   - PIN stored hashed; strict attempt limit with temporary lockout; the QR token alone never grants access.
3. **QR for identification:** the Pengelola can scan a nasabah's QR to select that nasabah when recording a deposit. This is lookup within the Pengelola's own tenant and creates no session for the nasabah.
4. **Account creation:** a Pengelola registers a bank sampah, which creates the tenant and the first Pengelola account. Nasabah accounts are created by the Pengelola with an initial password that the nasabah is prompted to change.
5. **Sessions:** short-lived access token (carries user id, role, bank_sampah_id) plus a rotating refresh token stored hashed server-side, revocable on logout and password change.
6. **Roles:** only Nasabah and Pengelola. Authorization checks role and tenant on every request; tenant id comes from the token, never request input.

## Alternatives considered
- QR + PIN only (as in BAB I): simple for residents but weak as the only method (PIN space is small, QR can be shared or photographed), and gives no recovery path. Kept only as an extra.
- Supabase Auth: rejected for portability (see ADR 0001).
- Email-only login: many residents lack email; username-based login works for them.

## Consequences
- Auth module owns users, credentials, QR tokens, refresh tokens.
- Password reset for Nasabah goes through the Pengelola; Pengelola recovery path (email-based) to be specified in M1 and may update this ADR.
- Tests must cover: rate limiting/lockout, QR token revocation, QR token alone cannot authenticate, tenant isolation across all auth paths.
- The project document's description (QR + PIN) differs from this decision; mention the change in the thesis write-up as a design refinement.
