# Plan: M3 — Deposits and Balance

Status: completed
Roadmap: `../ROADMAP.md` (M3)

## Ringkasan Hasil Eksekusi (Execution Summary)
Milestone M3 berhasil diselesaikan dengan hasil:
1. **Pencatatan Setoran Idempoten (`POST /api/v1/deposits`)**:
   - Pengelola mencatat setoran dengan `idempotency_key` dari client.
   - Replay request dengan `idempotency_key` yang sama mengembalikan data setoran sebelumnya (`HTTP 200 OK`) tanpa membuat record duplikat dan tanpa menduplikasi saldo nasabah.
2. **Snapshot Riwayat Harga & Pembulatan Integer Rupiah**:
   - Harga materi diambil dari `material_prices` pada waktu `client_recorded_at`.
   - Formula pembulatan half-up integer rupiah diterapkan: `(weight_grams * price_per_kg_idr + 500) / 1000`.
   - Perubahan harga di masa depan tidak mempengaruhi harga pada setoran lampau.
3. **Semantik Sinkronisasi Offline (`POST /api/v1/deposits/sync`)**:
   - Menerima batch setoran dalam format array untuk antrean sinkronisasi offline.
   - Mendeteksi dan menandai item baru sebagai `created` dan item replay sebagai `replayed`.
4. **Struk Digital (`GET /api/v1/deposits/:id/receipt`)**:
   - Menghasilkan rincian struk digital lengkap dengan profil bank sampah, nasabah, pengelola, nomor struk berformat `STR-<kode>-<timestamp>-<id>`, breakdown sampah, dan saldo terkini.
5. **Kalkulasi Saldo Terderivasi (`GET /api/v1/nasabah/me/balance` & `/:id/balance`)**:
   - Saldo selalu dihitung secara matematis dari `SUM(total_amount_idr)`.
6. **Isolasi Multi-Tenant & RBAC**:
   - Pengelola dan nasabah dari Tenant B tidak dapat melihat atau memodifikasi setoran, struk, atau saldo dari Tenant A.
   - Nasabah dari Tenant A hanya dapat melihat setoran miliknya sendiri.
7. **Pengujian**:
   - Unit test untuk pembulatan subtotal di `internal/deposit/service_test.go`.
   - Integration test alur setoran, idempotensi, struk, dan batch sync di `test/integration/deposit_test.go`.
   - Tenant isolation test untuk setoran dan saldo di `test/integration/tenant_isolation_test.go`.

## Tujuan (Goal)
Membangun modul pencatatan setoran sampah (*deposits*) dan kalkulasi saldo (*balance*) nasabah yang tangguh terhadap sinkronisasi offline:
1. **Pencatatan Setoran Idempoten**: Pengelola dapat mencatat setoran dengan `idempotency_key` dari client. Permintaan berulang (replay) menghasilkan tepat 1 transaksi dan mengembalikan data setoran yang sudah tersimpan.
2. **Snapshot Riwayat Harga (Historical Price Snapshotting)**: Harga per kg dan subtotal rupiah di-snapshot ke dalam baris setoran (`deposit_lines`) berdasarkan harga yang berlaku pada `client_recorded_at`.
3. **Pembulatan Integer Rupiah (Hard Rule #3)**: Formula pembulatan setengah ke atas integer `(weight_grams * price_per_kg_idr + 500) / 1000`.
4. **Semantik Sinkronisasi Offline**: Mendukung pengiriman setoran tunggal (`POST /api/v1/deposits`) maupun batch/bulk sync (`POST /api/v1/deposits/sync`).
5. **Struk Digital (Digital Receipt)**: Endpoint detail struk digital lengkap dengan profil bank sampah, nasabah, pengelola pencatat, rincian materi, total berat, total rupiah, dan saldo terkini.
6. **Kalkulasi Saldo Terderivasi (Derived Balance)**: Saldo nasabah selalu merupakan hasil penjumlahan seluruh entri setoran (`SUM(total_amount_idr)`).
7. **Isolasi Multi-Tenant & Otorisasi Peran**:
   - Pengelola hanya dapat melihat/mencatat setoran di bank sampah miliknya.
   - Nasabah hanya dapat melihat setoran dan saldo miliknya sendiri.

---

## Deliverables
- `docs/decisions/0007-deposit-idempotency-and-offline-sync.md`
- `db/migrations/000004_m3_deposits_balance.up.sql` & `down.sql`
- `db/queries/deposits.sql` & generated sqlc code
- `internal/deposit/model.go`, `repository.go`, `service.go`, `service_test.go`, `handler.go`
- Wiring in `cmd/api/main.go` and `internal/platform/router/router.go`
- Integration tests in `test/integration/deposit_test.go` and `test/integration/tenant_isolation_test.go`
- Updated `docs/DATA-MODEL.md` and `docs/PROJECT-CONTEXT.md`
