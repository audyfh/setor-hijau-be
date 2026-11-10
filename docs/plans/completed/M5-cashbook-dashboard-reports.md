# Plan: M5 — Cashbook, Dashboard, & Reports

Status: completed
Roadmap: `../ROADMAP.md` (M5)

## Ringkasan Hasil Eksekusi (Execution Summary)
Milestone M5 telah diselesaikan secara lengkap dengan seluruh penerimaan kriteria terpenuhi:
1. **Pencatatan Penjualan ke Pengepul (`sales_to_pengepul` & `sales_to_pengepul_lines`)**:
   - Pengelola mencatat penjualan sampah daur ulang terakumulasi ke pengepul/lapak pembeli (`POST /api/v1/cashbook/sales`).
   - Mendukung rincian multi-baris material dengan pembulatan setengah ke atas integer Rupiah: `(weight_grams * price_per_kg_idr + 500) / 1000`.
   - Secara atomik membuat mutasi kas masuk (`cash_in`) dengan kategori `'penjualan_pengepul'` pada buku kas.
2. **Penarikan Saldo Tunai Nasabah (`nasabah_withdrawals`)**:
   - Pengelola mencatat pencairan/penarikan saldo tabungan nasabah (`POST /api/v1/nasabah/:id/withdrawals`).
   - Proteksi saldo (overdraft protection): penarikan di atas saldo nasabah saat ini ditolak dengan `HTTP 422 Unprocessable Entity`.
   - Formula saldo terderivasi Domain Invariant #2: `current_balance_idr = SUM(deposits) - SUM(withdrawals)`.
   - Secara atomik membuat mutasi kas keluar (`cash_out`) dengan kategori `'penarikan_nasabah'` pada buku kas.
   - Nasabah dapat memeriksa riwayat penarikan miliknya sendiri (`GET /api/v1/nasabah/me/withdrawals`).
3. **Buku Kas Append-Only & Rekonsiliasi Keuangan (`cashbook_entries`)**:
   - Pencatatan transaksi kas masuk dan keluar (`POST /api/v1/cashbook/entries` & `GET /api/v1/cashbook/entries`).
   - Ringkasan kas (`GET /api/v1/cashbook/summary`): Total Kas Masuk, Total Kas Keluar, Saldo Kas di Tangan (*Cash on Hand*).
   - Rekonsiliasi keuangan (`GET /api/v1/cashbook/reconciliation`):
     - Memverifikasi total penjualan pengepul sama persis dengan kas masuk kategori penjualan.
     - Memverifikasi total penarikan nasabah sama persis dengan kas keluar kategori penarikan.
     - Menghitung kewajiban saldo nasabah yang belum ditarik vs kas fisik di tangan, serta surplus/defisit bersih.
4. **Dasbor Agregasi Bulanan (`internal/report`)**:
   - `GET /api/v1/dashboard/monthly?month=YYYY-MM`: Agregasi lengkap per bulan kalender:
     - Metrik Setoran: total transaksi, nasabah aktif menyetor, total berat (g & kg), total nilai rupiah, breakdown per kategori material.
     - Metrik Penjualan: total transaksi penjualan, total berat terjual, total pendapatan penjualan, breakdown per kategori material.
     - Metrik Keuangan: arus kas bersih, saldo kas di tangan, margin laba kotor operasional bank sampah.
     - Metrik Dampak: total estimasi emisi gas rumah kaca yang dicegah ($CO_2e$ avoided) dan breakdown per kategori (US EPA WARM v16).
5. **Ekspor Laporan Resmi (CSV & PDF)**:
   - `GET /api/v1/reports/monthly/csv?month=YYYY-MM`: Ekspor CSV berstandar internasional dengan UTF-8 BOM (`\xef\xbb\xbf`) agar langsung kompatibel tanpa *encoding glitch* di Microsoft Excel.
   - `GET /api/v1/reports/monthly/pdf?month=YYYY-MM`: Pembuatan dokumen PDF A4 resmi menggunakan library Go murni (`jung-kurt/gofpdf`) berprinsip Green Software Engineering (sangat ringan, cepat, tanpa headless Chromium/browser dependencies), mencakup kop resmi bank sampah, ringkasan eksekutif, tabel material & emisi karbon, dan kolom tanda tangan pengelola.
6. **Pengujian & Keamanan Multi-Tenant**:
   - Unit tests: kalkulasi subtotal dan parsing periode bulan serta validasi UTF-8 BOM CSV & header PDF.
   - Integration tests: `test/integration/cashbook_test.go` dan `test/integration/report_test.go` memverifikasi seluruh alur operasional, idempotensi, proteksi saldo, dan isolasi tenant secara ketat.

## Tujuan (Goal)
Mengimplementasikan modul pembukuan kas otomatis (Cashbook), pencatatan penjualan sampah ke pengepul, penarikan saldo nasabah (tarik tunai), rekonsiliasi keuangan, agregasi dasbor bulanan, serta ekspor laporan bulanan berformat CSV dan PDF:
1. Menyusun ADR 0009 di `docs/decisions/0009-cashbook-and-monthly-reports.md`.
2. Migrasi skema database `sales_to_pengepul`, `sales_to_pengepul_lines`, `nasabah_withdrawals`, dan `cashbook_entries`.
3. Memperbarui kalkulasi saldo nasabah terderivasi di `internal/deposit` agar memperhitungkan penarikan tunai (`deposits - withdrawals`).
4. Implementasi modul internal `internal/cashbook` (penjualan ke pengepul, penarikan saldo nasabah, mutasi kas, rekonsiliasi finansial).
5. Implementasi modul internal `internal/report` (agregasi dashboard bulanan, ekspor CSV dengan UTF-8 BOM, dan ekspor PDF berbasis `gofpdf`).
6. Pengujian unit kalkulasi rekonsiliasi dan format laporan.
7. Pengujian integrasi alur end-to-end buku kas, penjualan, penarikan, ekspor laporan, dan proteksi isolasi multi-tenant.

## Deliverables
- `docs/decisions/0009-cashbook-and-monthly-reports.md`
- `docs/DATA-MODEL.md` (pembaruan skema M5)
- `db/migrations/000006_m5_cashbook_reports.up.sql` & `down.sql`
- `db/queries/cashbook.sql` & `db/queries/reports.sql` & sqlc generated code
- `internal/cashbook/model.go`, `repository.go`, `service.go`, `handler.go`, `service_test.go`
- `internal/report/model.go`, `repository.go`, `service.go`, `handler.go`, `csv.go`, `pdf.go`, `service_test.go`
- Update `internal/deposit/` (saldo dikurangi penarikan)
- Wiring routes di `internal/platform/router/router.go` dan `cmd/api/main.go`
- Integration tests di `test/integration/cashbook_test.go` dan `test/integration/report_test.go`
- Update `docs/PROJECT-CONTEXT.md` dan pemindahan plan ke `completed/`
