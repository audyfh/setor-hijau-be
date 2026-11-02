# Plan: M2 — Nasabah and Master Data

Status: completed
Roadmap: `../ROADMAP.md` (M2)

## Tujuan (Goal)
Membangun modul manajemen Nasabah dan Master Data (Jenis Sampah & Riwayat Harga) sesuai spesifikasi domain Setor Hijau:
1. **Nasabah CRUD oleh Pengelola**:
   - Pengelola mendaftarkan nasabah baru di Bank Sampah-nya (termasuk generate token QR awal yang unguessable & opaque, serta default password/PIN).
   - Pengelola dapat melihat daftar nasabah (dengan pencarian, filter, paginasi), melihat profil lengkap nasabah, mengedit data nasabah, mengaktifkan/menonaktifkan akun, serta meregenerasi token QR jika kartu fisik hilang.
   - Pengelola dapat melakukan reset password atau PIN nasabah (sesuai ADR 0002).
2. **Autentikasi Nasabah**:
   - **Login Primer (Identifier + Password)**: Nasabah login menggunakan username + password + kode bank sampah (atau email jika terdaftar), menerima short-lived JWT access token dan refresh token.
   - **Login Sekunder (QR Token + PIN)**: Nasabah login menggunakan QR token fisik + PIN 6 digit. QR token sendiri **tidak dapat** mengautentikasi tanpa PIN yang benar (kriteria akseptasi wajib).
   - Rate limiting & lockout proteksi brute-force PIN (misal 5 kali gagal -> lockout sementara).
3. **Pengelola QR Lookup**:
   - Pengelola dapat memindai QR token nasabah untuk memilih nasabah saat pencatatan setoran (lookup cepat dalam scope tenant Pengelola, tanpa membuat sesi nasabah).
4. **Master Data Jenis Sampah & Riwayat Harga (Material Types & Price History)**:
   - Pengelola dapat membuat, membaca, memperbarui jenis sampah (*material types*).
   - Pengelola dapat memperbarui harga per kg (`price_per_kg_idr` dalam integer rupiah, Hard Rule #3).
   - Riwayat harga disimpan secara append-only menggunakan rentang `effective_from` dan `effective_to` agar perubahan harga di masa depan **tidak mengubah** nilai setoran terdahulu.
   - Nasabah dan Pengelola dapat melihat daftar jenis sampah aktif beserta harga berlakunya.

---

## Keputusan Arsitektur & Desain (Architectural Decisions)

1. **ADR 0006: QR Code Token & Nasabah PIN Security**:
   - Token QR adalah string acak kriptografis (32 bytes hex/base64url) yang unguessable dan tidak memuat ID nasabah atau nomor urut.
   - Di database disimpan dalam bentuk hash SHA-256 (`qr_token_hash`).
   - PIN nasabah adalah 6 digit angka, disimpan ter-hash menggunakan Argon2id (`pin_hash`).
   - Rate limit ketat diterapkan pada kegagalan verifikasi PIN (`failed_pin_attempts` dan `locked_until`).
2. **Multi-Tenant Nasabah Username Resolution**:
   - Pengelola unik secara global (email/username platform-wide).
   - Username nasabah unik per `bank_sampah_id`. Untuk login username + password, request menyertakan identifier `bank_sampah_code` (slug) atau email unik. Untuk QR + PIN login, token QR hash yang bersifat unguessable & globally unique langsung merujuk ke profil nasabah dan tenant-nya.
3. **Price History Immutability**:
   - Tabel `material_prices` menyimpan `effective_from` dan `effective_to`. Ketika harga baru dibuat, harga lama ditutup (`effective_to = now()`), dan record harga baru dibuat (`effective_from = now()`, `effective_to = NULL`).

---

## Rincian Perubahan Database & Query

### File Migrasi: `000003_m2_nasabah_master_data`
1. `nasabah_profiles`:
   - `user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE`
   - `bank_sampah_id UUID NOT NULL REFERENCES bank_sampah(id) ON DELETE CASCADE`
   - `qr_token_hash VARCHAR(64) NOT NULL UNIQUE`
   - `pin_hash TEXT`
   - `failed_pin_attempts INT NOT NULL DEFAULT 0`
   - `locked_until TIMESTAMPTZ`
   - `rw VARCHAR(10) NOT NULL`
   - `rt VARCHAR(10)`
   - `address_notes TEXT`
   - `registered_by UUID REFERENCES users(id) ON DELETE SET NULL`
   - `created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP`
   - `updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP`
2. `material_types`:
   - `id UUID PRIMARY KEY DEFAULT gen_random_uuid()`
   - `bank_sampah_id UUID NOT NULL REFERENCES bank_sampah(id) ON DELETE CASCADE`
   - `name VARCHAR(100) NOT NULL`
   - `category VARCHAR(50) NOT NULL`
   - `unit VARCHAR(20) NOT NULL DEFAULT 'kg'`
   - `description TEXT`
   - `is_active BOOLEAN NOT NULL DEFAULT TRUE`
   - `created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP`
   - `updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP`
   - Constraint: `UNIQUE (bank_sampah_id, name)`
3. `material_prices`:
   - `id UUID PRIMARY KEY DEFAULT gen_random_uuid()`
   - `material_type_id UUID NOT NULL REFERENCES material_types(id) ON DELETE CASCADE`
   - `bank_sampah_id UUID NOT NULL REFERENCES bank_sampah(id) ON DELETE CASCADE`
   - `price_per_kg_idr BIGINT NOT NULL CHECK (price_per_kg_idr >= 0)`
   - `effective_from TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP`
   - `effective_to TIMESTAMPTZ`
   - `created_by UUID REFERENCES users(id) ON DELETE SET NULL`
   - `created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP`

### sqlc Queries
- `db/queries/nasabah.sql`: CRUD queries untuk nasabah dan profil, lookup by QR hash, filter tenant.
- `db/queries/materials.sql`: CRUD queries untuk material types, price insertion, query current active price, price history.
- `db/queries/auth.sql`: Extend query nasabah login by bank_sampah_id + username, and lookup by QR hash.

---

## Struktur Modul Go

1. `internal/nasabah`:
   - `model.go`: DTO & domain models untuk Nasabah, profil, request/response CRUD, QR lookup, PIN management.
   - `repository.go`: Database operations dengan sqlc wrapper dan transaksi.
   - `service.go`: Business logic pendaftaran nasabah (buat user + profile + generate QR token), regenerate QR token, set PIN, lookup QR oleh pengelola.
   - `handler.go`: HTTP handler Gin untuk endpoint nasabah.
2. `internal/material`:
   - `model.go`: DTO & domain models untuk Jenis Sampah dan Harga.
   - `repository.go`: Database operations dengan sqlc wrapper.
   - `service.go`: Business logic pembuatan jenis sampah dengan harga awal, pembaruan harga (price history), query list materi aktif dengan harga terkini.
   - `handler.go`: HTTP handler Gin untuk endpoint materials.
3. `internal/auth`:
   - Penambahan method `LoginWithQR(ctx, qrToken, pin)` untuk autentikasi QR + PIN nasabah.
   - Dukungan `bank_sampah_code` pada endpoint login primer.
   - Validasi ketat: penolakan jika hanya menyertakan QR token tanpa PIN yang valid.
4. `internal/platform/router`:
   - Registrasi rute `/api/v1/nasabah` dan `/api/v1/materials`.

---

## Rincian Endpoint HTTP

### Nasabah Management & Auth
- `POST /api/v1/auth/login` (Public, Rate Limited): Login username + password (mendukung Pengelola & Nasabah).
- `POST /api/v1/auth/login/qr` (Public, Rate Limited): Login QR token + PIN (Nasabah saja).
- `POST /api/v1/nasabah` (Auth Pengelola): Daftarkan Nasabah baru (mengembalikan data nasabah dan raw QR token).
- `GET /api/v1/nasabah` (Auth Pengelola): Daftar Nasabah (pagination, search, filter RT/RW).
- `GET /api/v1/nasabah/:id` (Auth Pengelola): Detail Nasabah.
- `PUT /api/v1/nasabah/:id` (Auth Pengelola): Edit profil Nasabah & status aktif.
- `POST /api/v1/nasabah/:id/regenerate-qr` (Auth Pengelola): Generate ulang QR token (invalidasi yang lama).
- `POST /api/v1/nasabah/:id/reset-password` (Auth Pengelola): Set password baru nasabah.
- `POST /api/v1/nasabah/:id/reset-pin` (Auth Pengelola): Set PIN baru nasabah.
- `POST /api/v1/nasabah/lookup-qr` (Auth Pengelola): Lookup identitas nasabah via QR token dalam tenant yang sama (tanpa membuat sesi login).
- `GET /api/v1/nasabah/me` (Auth Nasabah): Profil nasabah sendiri.
- `PUT /api/v1/nasabah/me/pin` (Auth Nasabah): Set / ganti PIN mandiri.
- `PUT /api/v1/nasabah/me/password` (Auth Nasabah): Ganti password mandiri.

### Master Data Jenis Sampah & Harga
- `POST /api/v1/materials` (Auth Pengelola): Tambah jenis sampah baru beserta harga awal per kg.
- `GET /api/v1/materials` (Auth Pengelola & Nasabah): List jenis sampah dengan harga aktif terkini (`?active_only=true|false`).
- `GET /api/v1/materials/:id` (Auth Pengelola & Nasabah): Detail jenis sampah beserta harga saat ini.
- `PUT /api/v1/materials/:id` (Auth Pengelola): Update metadata jenis sampah (nama, kategori, deskripsi, is_active).
- `POST /api/v1/materials/:id/prices` (Auth Pengelola): Update harga jenis sampah (membuat riwayat harga baru).
- `GET /api/v1/materials/:id/prices` (Auth Pengelola): Riwayat perubahan harga jenis sampah.

---

## Rencana Pengujian (Verification Plan)

1. **Unit Tests**:
   - Validasi DTO, PIN format (6 digit), sanitasi input.
   - Logika status lockout dan rate-limit PIN.
2. **Integration Tests (`test/integration/`)**:
   - **QR & PIN Security Test**:
     - Memastikan QR token saja tanpa PIN **gagal mengautentikasi** (401).
     - Memastikan PIN salah ditolak dan setelah 5 kali salah akun terkunci sementara.
     - Memastikan login QR + PIN yang benar berhasil mengeluarkan token JWT dan refresh token.
     - Memastikan token QR yang di-regenerate membatalkan QR token lama.
   - **Tenant Isolation Test (Hard Rule #1)**:
     - Pengelola Tenant A tidak dapat melihat, mengubah, atau meregenerasi nasabah Tenant B.
     - Pengelola Tenant A memindai QR nasabah Tenant B mengembalikan 404 (Not Found).
     - Material types dan prices Tenant A tidak bocor ke Tenant B.
   - **Price History Invariant Test (Hard Rule #3 & Domain Invariant #4)**:
     - Mengubah harga jenis sampah mencatat record baru di `material_prices` dengan `effective_from` baru.
     - Riwayat harga lampau tetap utuh dan harga aktif adalah harga terkini (`effective_to IS NULL`).
3. **Make Check**:
   - `make lint`, `make test`, `make sqlc` diff check lulus bersih.
