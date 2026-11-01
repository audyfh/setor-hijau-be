# Plan: M1 — Tenancy and Auth

Status: completed
Roadmap: `../ROADMAP.md` (M1)

## Tujuan (Goal)
Membangun fondasi multi-tenancy dan autentikasi untuk Setor Hijau:
1. Registrasi Bank Sampah (tenant) beserta akun Pengelola pertama.
2. Autentikasi primer (identifier + password) untuk Pengelola dengan hashing Argon2id.
3. Mekanisme token sesi (short-lived JWT access token + rotasi refresh token yang disimpan ter-hash di DB).
4. Middleware tenant context & role-based access control (RBAC) yang mengekstrak `bank_sampah_id` dari token dan menyuntikkannya ke context (Hard Rule #1).
5. Rate limiting untuk endpoint login guna mencegah brute-force.
6. Harness pengujian isolasi tenant (*tenant isolation test harness*) yang dapat digunakan kembali untuk modul-modul berikutnya.
7. Dokumentasi schema awal di `docs/DATA-MODEL.md` dan ADR terkait strategi multi-tenancy/RLS (ADR 0004) serta recovery akun (ADR 0005).

---

## Keputusan & Usulan ADR (Decisions & Proposed ADRs)

1. **ADR 0004: Multi-tenancy Isolation Strategy (Application-Level vs PostgreSQL RLS)**:
   - Disepakati **Application-Enforced Tenancy** via `sqlc` queries + context injection sebagai lapisan primer wajib, didukung oleh foreign key constraint dan index komposit `(bank_sampah_id, id)`. Pengujian isolasi tenant otomatis di CI memastikan tidak ada data yang bocor antar tenant.
2. **ADR 0005: Pengelola Account Recovery & Password Reset Flow**:
   - Dibuat tabel `password_reset_tokens` (hashed token, user_id, expires_at 15 menit, used_at) dan endpoint `POST /auth/forgot-password` serta `POST /auth/reset-password`. Abstraksi interface `EmailSender` menggunakan `LogEmailSender` di dev/test.
3. **Dokumentasi Skema `docs/DATA-MODEL.md`**:
   - Ditetapkan konvensi tipe data: UUIDv4 untuk ID, integer rupiah untuk uang, integer gram untuk berat, integer gram CO2e untuk emisi karbon, dan UTC timestamptz.

---

## Langkah Berurutan (Sequential Steps)

### Fase 1: Dokumentasi Arsitektur & Data Model
- [x] Buat ADR 0004 (`docs/decisions/0004-multitenancy-isolation.md`) dan ADR 0005 (`docs/decisions/0005-pengelola-account-recovery.md`).
- [x] Buat `docs/DATA-MODEL.md` yang memuat perancangan skema relasional lengkap (tabel M1, konvensi tipe data integer, indeks, dan relasi tenant).

### Fase 2: Skema Basis Data & sqlc Queries
- [x] Buat file migrasi `db/migrations/000002_m1_tenancy_auth.up.sql` dan `000002_m1_tenancy_auth.down.sql`:
  - Tabel `bank_sampah`, `users`, `refresh_tokens`, `password_reset_tokens`.
- [x] Tulis query SQL di `db/queries/tenants.sql` dan `db/queries/auth.sql`.
- [x] Jalankan `make sqlc` untuk menghasilkan kode Go repository di `internal/platform/db/sqlc`.

### Fase 3: Fondasi Platform & Security Utilities
- [x] Modul security di `internal/platform/auth`:
  - Hashing Argon2id (`HashPassword`, `VerifyPassword`) dengan parameter OWASP.
  - JWT Token manager (generate token, parse claims).
  - Opaque random string generator untuk Refresh Token dan Reset Token (disimpan dengan SHA-256 hash).
- [x] Middleware di `internal/platform/middleware`:
  - `TenantContextMiddleware`: Validasi JWT `Authorization: Bearer <token>`, simpan `bank_sampah_id`, `user_id`, dan `role` ke request context.
  - `RequireRole(roles ...string)`: Memastikan hanya role yang diizinkan yang dapat mengakses endpoint tertentu.
  - `RateLimiter`: In-memory rate limiting untuk endpoint autentikasi berbasis IP dan identifier.

### Fase 4: Implementasi Modul Tenant & Auth
- [x] Modul Tenant (`internal/tenant`):
  - `repository.go`: Adapter ke query sqlc `bank_sampah`.
  - `service.go`: Bisnis logic registrasi Bank Sampah baru beserta pembuatan akun Pengelola pertama secara transaksional (`atomic`).
  - `handler.go`: Endpoint `POST /api/v1/tenants/register` dan `GET /api/v1/tenants/current`.
- [x] Modul Auth (`internal/auth`):
  - `repository.go`: Adapter ke query sqlc `users`, `refresh_tokens`, `password_reset_tokens`.
  - `service.go`: Login, RefreshToken, Logout, ForgotPassword, ResetPassword, GetMe.
  - `handler.go`: Endpoint auth v1.
- [x] Routing & Wiring di `cmd/api/main.go` dan `internal/platform/router`.

### Fase 5: Harness Pengujian Isolasi Tenant & Verifikasi
- [x] Buat harness pengujian integrasi yang reusable di `test/integration`:
  - Setup database pengujian dengan migrations runner dan table cleanups.
- [x] Tulis test suites:
  - **Tenant Isolation Test**: Pengelola Tenant A tidak dapat membaca/mengubah data milik Tenant B.
  - **Auth & Token Lifecycle Test**: Registrasi -> Login -> Akses Endpoint Terproteksi -> Refresh Token -> Rotasi Refresh Token -> Logout.
  - **Rate Limiting Test**: Uji pembatasan brute-force pada endpoint login.
  - **Password Security Test**: Argon2id hashing verification dan penolakan password yang salah.
- [x] Jalankan `make check` (linting `golangci-lint`, unit + integration tests, dan cek `git diff --exit-code internal/platform/db/sqlc`).
- [x] Perbarui status di `docs/PROJECT-CONTEXT.md` dan arsipkan plan jika selesai.

---

## Kriteria Selesai (Acceptance Criteria)

1. **Registrasi Bank Sampah & Pengelola**:
   - `POST /api/v1/tenants/register` berhasil membuat record `bank_sampah` dan user Pengelola pertama dalam 1 transaksi DB.
2. **Login & Session Management**:
   - Pengelola dapat login dengan username/email + password, menerima JWT access token dan refresh token.
   - Rotasi refresh token berfungsi dan token lama langsung di-revoke.
   - Logout berhasil me-revoke refresh token sehingga tidak dapat digunakan kembali.
   - Password disimpan menggunakan Argon2id hash; password teks asli tidak pernah di-log.
3. **Isolasi Tenant Terbukti (Hard Rule #1)**:
   - Request ke endpoint terproteksi wajib menyertakan token valid.
   - Context membawa `bank_sampah_id` dari token; query database memfilter secara ketat dengan `bank_sampah_id`.
   - Test otomatis membuktikan Tenant A tidak dapat mengakses resource Tenant B.
4. **Rate Limiting**:
   - Endpoint login menolak request berlebih (HTTP 429 Too Many Requests) jika terjadi upaya login berulang kali.
5. **Kualitas & Standar Proyek**:
   - `make check` lulus tanpa error (lint bersih, unit & integration test lulus, sqlc diff bersih).
   - Dokumen `docs/DATA-MODEL.md`, ADR 0004, dan ADR 0005 lengkap dan terdokumentasi.

---

## Risiko & Mitigasi (Risks & Mitigations)

| Risiko | Dampak | Mitigasi |
|---|---|---|
| **Kebocoran Data Antar Tenant (*Data Leakage*)** | Fatal bagi sistem multi-tenant. | 1. Terapkan Hard Rule #1: Semua query tenant-scoped memiliki parameter wajib `bank_sampah_id` dari context.<br>2. Unique composite constraints di DB `(bank_sampah_id, ...)`.<br>3. Automated tenant-isolation test harness. |
| **Koneksi Pooler Supabase & Session State** | State bisa bocor jika ada query tanpa transaksi saat pakai `SET LOCAL`. | Gunakan pendekatan Application-Enforced Isolation di sqlc query layer. |
| **Overhead Komputasi Argon2id** | Konsumsi CPU/Memory tinggi saat traffic login bersamaan. | Konfigurasi parameter Argon2id sesuai rekomendasi OWASP, dipadu dengan rate limiter. |
| **Penyalahgunaan Refresh Token** | Penyerang menggunakan refresh token yang dicuri untuk memperpanjang sesi. | 1. Refresh token disimpan SHA-256 hash di database.<br>2. Rotasi token setiap kali refresh (refresh token hanya valid 1 kali pakai).<br>3. Penyediaan endpoint revoke (logout). |
| **Ketergantungan Layanan Email Eksternal untuk Recovery** | Memerlukan kredensial SMTP/API eksternal. | Desain interface `EmailSender` dengan implementasi log-based mock di dev/test environment. |
