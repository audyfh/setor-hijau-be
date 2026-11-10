# Plan: M6 — Net Impact Meter & Public Method Page

Status: completed
Roadmap: `../ROADMAP.md` (M6)

## Tujuan (Goal)
Mengimplementasikan modul pengukuran jejak karbon digital aplikasi berbasis Sustainable Web Design Model (SWDM) v4, penghitungan Net Impact Meter (emisi dihindari minus jejak digital), serta penyediaan endpoint publik transparansi ilmiah "Metode & Sumber":
1. Menyusun ADR 0010 di `docs/decisions/0010-digital-footprint-and-net-impact-meter.md`.
2. Memperbarui `docs/CARBON-METHOD.md` dan `docs/DATA-MODEL.md`.
3. Migrasi skema database `digital_footprint_measurements`.
4. Implementasi modul internal `internal/impact` (telemetri data transfer, perhitungan integer micrograms SWDM v4, agregasi Net Impact platform dan tenant, serta penyajian publik metode & sumber).
5. Pengujian unit perhitungan SWDM v4 dan formula Net Impact (`internal/impact/service_test.go`).
6. Pengujian integrasi endpoint publik `/public/methods`, `/public/impact`, `/public/impact/tenants/:code`, telemetri, dan isolasi tenant (`test/integration/impact_test.go`).

## Deliverables
- `docs/decisions/0010-digital-footprint-and-net-impact-meter.md` (ADR 0010 disetujui)
- `docs/CARBON-METHOD.md` (penambahan Section 8 SWDM v4 & Section 9 Net Impact Meter)
- `docs/DATA-MODEL.md` (Entitas 16: `digital_footprint_measurements`)
- `db/migrations/000007_m6_net_impact.up.sql` & `down.sql` (migrasi diterapkan)
- `db/queries/impact.sql` & sqlc generated code di `internal/platform/db/sqlc/impact.sql.go`
- `internal/impact/model.go`, `repository.go`, `service.go`, `handler.go`, `service_test.go`
- Wiring routes di `internal/platform/router/router.go`, `cmd/api/main.go`, dan `test/integration/testhelper.go`
- Integration tests di `test/integration/impact_test.go` (100% lulus)
- Update `docs/PROJECT-CONTEXT.md` dan `docs/plans/ROADMAP.md`
