# ADR 0005: Pengelola Account Recovery

Status: Accepted, 2026-11-01

## Context
ADR 0002 specifies that:
- Nasabah account credentials and password resets are managed directly by the Pengelola in person, as residents often lack email addresses.
- Pengelola require a self-service account recovery path in case of forgotten passwords.
- Cloud vendor neutrality (Hard Rule #6) prohibits embedding vendor-specific SDKs (e.g., AWS SES, Firebase Auth, Supabase Auth) into domain logic.

## Decision
1. **Token-Based Recovery Flow**:
   - Provide `POST /api/v1/auth/forgot-password` (accepting email).
   - If the email belongs to an active Pengelola account, generate a cryptographically secure random token (32 bytes, URL-safe base64 / hex).
   - Store the SHA-256 hash of the token in `password_reset_tokens` with an expiration time of 15 minutes (`expires_at`), referencing `user_id`.
   - Provide `POST /api/v1/auth/reset-password` (accepting token and new password). Upon successful reset, verify the token hash, check expiry, mark the token as used (`used_at = NOW()`), update `password_hash` using Argon2id, and revoke all existing active refresh tokens for that user.
   - For security against email enumeration, `POST /api/v1/auth/forgot-password` returns a generic 200 OK success message regardless of whether the email was registered.
2. **Email Sender Abstraction**:
   - Define a domain service interface `EmailSender` (`SendPasswordReset(ctx, toEmail, resetToken, resetURL) error`).
   - In local development, testing, and initial milestone environments, implement a `LogEmailSender` that outputs the reset link to structured JSON logs (`slog.Info`).
   - Production SMTP or webhook delivery can be plugged in via configuration without altering the core authentication service.

## Consequences
- No external email provider is required to run tests or local development.
- Secure, single-use, time-limited reset tokens prevent replay attacks.
- Nasabah accounts remain excluded from self-service reset; only Pengelola accounts with registered emails can initiate recovery.
