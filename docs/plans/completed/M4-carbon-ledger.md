# Plan: M4 — Carbon Ledger

Status: completed
Roadmap: `../ROADMAP.md` (M4)

## Ringkasan Hasil Eksekusi (Execution Summary)
Milestone M4 berhasil diselesaikan secara menyeluruh dengan hasil sebagai berikut:
1. **Dokumentasi Metodologi Ilmiah & Keputusan Arsitektur**:
   - `docs/CARBON-METHOD.md`: Mendokumentasikan basis ilmiah US EPA WARM v16, tabel konversi 7 material default (Plastik PET, HDPE, Kertas Karton/Kardus, Kertas Campur, Logam Aluminium, Kaca, Logam Besi/Baja), konversi faktor short ton ke metric ton, representasi integer PPM ($10^6$), formula half-up integer emisi, asumsi, limitasi, dan 14 hand-verified test cases.
   - `docs/decisions/0008-carbon-ledger-and-emission-factors.md`: Menetapkan keputusan tabel faktor emisi terversi tingkat platform, buku besar karbon append-only, integer PPM, dan relasi deposit.
2. **Skema Basis Data & Migrasi (`000005_m4_carbon_ledger`)**:
   - Tabel `emission_factors`: Platform-wide, terversi (`version = 'EPA_WARM_V16_2024'`), menyimpan faktor emisi dalam integer PPM per gram (`co2e_factor_ppm`).
   - Penambahan relasi `emission_factor_id` pada `material_types`.
   - Tabel `carbon_ledger`: Tenant-scoped (`bank_sampah_id`), append-only (`entry_type = 'deposit' | 'reversal'`), mencatat `deposit_line_id`, `nasabah_id`, `weight_grams`, `co2e_factor_ppm_snapshot`, `co2e_grams`, `source_citation`, dan `notes`.
   - Enforce check constraints: immutability dan larangan modifikasi/penghapusan.
3. **Modul `internal/carbon`**:
   - `model.go`: Domain model untuk entri karbon, ringkasan tabungan emisi, faktor emisi, dan reversing entries.
   - `repository.go`: Query terisolasi per tenant via sqlc (`db/queries/carbon.sql`).
   - `service.go`: Logika kalkulasi pembulatan half-up integer PPM `(weight_grams * co2e_factor_ppm + 500_000) / 1_000_000`, agregasi tabungan nasabah, pembacaan riwayat, pembuatan reversal, dan validasi.
   - `service_test.go`: 14 unit test berbasis hand-verified calculation dari EPA WARM v16 (100% lolos).
   - `handler.go`: Endpoint REST API:
     - `GET /api/v1/carbon/factors` (daftar faktor emisi terversi)
     - `GET /api/v1/carbon/ledger` (riwayat jurnal karbon tenant)
     - `GET /api/v1/carbon/nasabah/:id/savings` (akumulasi emisi dicegah per nasabah)
     - `GET /api/v1/carbon/me/savings` (akumulasi emisi nasabah login)
     - `POST /api/v1/carbon/reversals` (pembuatan reversing entry dengan nilai negatif dan catatan)
4. **Integrasi Atomik dengan Setoran (`internal/deposit`) & Materi (`internal/material`)**:
   - Pada saat setoran dicatat (`POST /api/v1/deposits`), baris entri `carbon_ledger` secara atomik dibuat dalam transaksi database yang sama untuk setiap materi yang memiliki `emission_factor_id`.
   - Struk digital (`GET /api/v1/deposits/:id/receipt`) menyertakan estimasi emisi per baris materi (`estimated_co2e_grams`, `co2e_factor_ppm`, `emission_source`) dan total emisi yang dicegah (`total_estimated_co2e_grams`).
   - Penanganan idempotensi: replay setoran tidak menduplikasi entri buku besar karbon.
5. **Pengujian Integrasi & Keamanan Multi-Tenant**:
   - `test/integration/carbon_test.go`:
     - Alur end-to-end setoran sampah plastik & kertas menghasilkan jurnal karbon dan memperbarui tabungan emisi nasabah.
     - Struk digital memuat rincian emisi yang akurat.
     - Replay request idempoten tidak menggandakan entri karbon ledger.
     - Pembuatan reversing entry berhasil mengurangi akumulasi tabungan karbon secara matematis.
     - Pengujian isolasi multi-tenant: Pengelola dan Nasabah dari Bank Sampah B tidak dapat membaca atau mengakses ledger/tabungan Bank Sampah A.

## Tujuan (Goal)
Mengimplementasikan modul pencatatan jejak karbon terdesentralisasi (Carbon Ledger) berbasis US EPA WARM v16 dengan prinsip append-only, tanpa float, isolasi multi-tenant, serta transparansi sumber dan auditabilitas penuh:
1. Menyusun dokumen metodologi ilmiah di `docs/CARBON-METHOD.md`.
2. Menyusun ADR 0008 di `docs/decisions/0008-carbon-ledger-and-emission-factors.md`.
3. Migrasi skema database `emission_factors`, `material_types.emission_factor_id`, dan `carbon_ledger`.
4. Implementasi modul internal `internal/carbon` (kalkulasi emisi integer half-up, per-nasabah savings, riwayat jurnal, dan reversing entries).
5. Integrasi atomik pencatatan setoran (`internal/deposit`) dengan `carbon_ledger` dan penyajian rincian emisi pada struk digital.
6. Pengujian unit hand-verified calculation dan pengujian integrasi lengkap.

## Deliverables
- `docs/CARBON-METHOD.md`
- `docs/decisions/0008-carbon-ledger-and-emission-factors.md`
- `db/migrations/000005_m4_carbon_ledger.up.sql` & `down.sql`
- `db/queries/carbon.sql` & generated sqlc code
- `internal/carbon/model.go`, `repository.go`, `service.go`, `service_test.go`, `handler.go`
- Update `internal/material/` dan `internal/deposit/`
- Wire router di `internal/platform/router/router.go` dan `cmd/api/main.go`
- Integration tests di `test/integration/carbon_test.go`
- Update `docs/DATA-MODEL.md` dan `docs/PROJECT-CONTEXT.md`
